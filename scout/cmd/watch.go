package scoutcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awesome-gocui/gocui"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/fatih/color"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers/nagioswatcher"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/logger"
	"github.com/choria-io/go-choria/scout/stream"
)

type WatchCommand struct {
	identity     string
	check        string
	perf         bool
	longestCheck int
	longestId    int
	statePattern string
	history      time.Duration
	nc           choria.Connector

	transEph *stream.Ephemeral
	stateEph *stream.Ephemeral

	status    map[string]map[string]string
	vwBuffers map[string][]string

	logger.Logrus
	sync.Mutex
}

func NewWatchCommand(idf string, checkf string, perf bool, history time.Duration, nc choria.Connector, log *logrus.Entry) (*WatchCommand, error) {
	w := &WatchCommand{
		identity:  idf,
		check:     checkf,
		perf:      perf,
		history:   history,
		nc:        nc,
		Logrus:    log,
		status:    make(map[string]map[string]string),
		vwBuffers: make(map[string][]string),
	}

	return w, nil
}

func (w *WatchCommand) Run(ctx context.Context, wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	lctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if w.history > time.Hour {
		return fmt.Errorf("maximum history that can be fetched is 1 hour")
	}

	gui, err := w.setupWindows()
	if err != nil {
		return err
	}
	defer gui.Close()

	transitions := make(chan *nats.Msg, 1000)
	states := make(chan *nats.Msg, 1000)

	go func() {
		var m *nats.Msg

		for {
			select {
			case m = <-transitions:
				w.handleTransition(m, gui)
			case m = <-states:
				w.handleState(m, gui)
			case <-ctx.Done():
				return
			}

			// no history means no jetstream
			if m.Reply == "" {
				continue
			}

			m.Ack()
		}
	}()

	if w.history > 0 {
		err = w.subscribeJetStream(lctx, transitions, states)
	} else {
		err = w.subscribeDirect(transitions, states)
	}
	if err != nil {
		return err
	}

	err = gui.MainLoop()
	if err != gocui.ErrQuit {
		return err
	}

	cancel()
	w.nc.Close()

	return nil
}

func (w *WatchCommand) dataFromCloudEventJSON(j []byte) ([]byte, error) {
	event := cloudevents.NewEvent("1.0")
	err := event.UnmarshalJSON(j)
	if err != nil {
		return nil, err
	}

	data, err := event.DataBytes()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (w *WatchCommand) handleTransition(m *nats.Msg, gui *gocui.Gui) {
	if m == nil {
		return
	}

	data, err := w.dataFromCloudEventJSON(m.Data)
	if err != nil {
		w.Errorf("could not parse cloud event: %s", err)
		return
	}

	transition := &machine.TransitionNotification{}
	err = json.Unmarshal(data, transition)
	if err != nil {
		w.Errorf("Could not decode received transition message: %s: %s", string(data), err)
		return
	}

	if w.identity != "" && !strings.Contains(transition.Identity, w.identity) {
		return
	}
	if w.check != "" && !strings.Contains(transition.Machine, w.check) {
		return
	}

	w.transEph.SetResumeSequence(m)

	w.Lock()
	defer w.Unlock()

	w.updateView(gui, "Transitions", true, func(o io.Writer, _ *gocui.View) {
		fmt.Fprintf(o, "%s %-20s %s => %s %s\n",
			time.Unix(transition.Timestamp, 0).Format("15:04:05"),
			transition.Identity,
			w.colorizeState(transition.FromState),
			w.colorizeState(transition.ToState),
			transition.Machine)
	})
}

func (w *WatchCommand) colorizeState(state string) string {
	switch state {
	case "OK":
		return color.GreenString("OK  ")
	case "WARNING":
		return color.YellowString("WARN")
	case "CRITICAL":
		return color.RedString("CRIT")
	case "UNKNOWN":
		return color.HiWhiteString("UNKN")
	default:
		return color.CyanString(state)
	}
}

func (w *WatchCommand) handleState(m *nats.Msg, gui *gocui.Gui) {
	if m == nil {
		return
	}

	data, err := w.dataFromCloudEventJSON(m.Data)
	if err != nil {
		w.Errorf("could not parse cloud event: %s", err)
		return
	}

	var state nagioswatcher.StateNotification
	err = json.Unmarshal(data, &state)
	if err != nil {
		w.Errorf("%s", err)
		return
	}

	if w.identity != "" && !strings.Contains(state.Identity, w.identity) {
		return
	}
	if w.check != "" && !strings.Contains(state.Machine, w.check) {
		return
	}

	output := strings.Split(state.Output, "|")
	w.stateEph.SetResumeSequence(m)

	w.Lock()
	defer w.Unlock()

	w.updateStatus(gui, &state)

	update := false
	if w.longestCheck < len(state.Machine) {
		w.longestCheck = len(state.Machine)
		update = true
	}

	if w.longestId < len(state.Identity) {
		w.longestId = len(state.Identity)
		update = true
	}

	if w.statePattern == "" || update {
		w.statePattern = "%s %s %" + strconv.Itoa(w.longestId) + "s %" + strconv.Itoa(w.longestCheck) + "s: "
	}

	w.updateView(gui, "Checks", true, func(o io.Writer, _ *gocui.View) {
		pre := fmt.Sprintf(w.statePattern, time.Unix(state.Timestamp, 0).Format("15:04:05"), w.colorizeState(state.Status), state.Identity, state.Machine)
		line := pre + output[0]
		fmt.Fprintln(o, line)

		if w.perf {
			for _, p := range state.PerfData {
				fmt.Fprintf(o, "%-"+strconv.Itoa(len(pre)-10)+"s %s = %v %s\n", "", p.Label, p.Value, p.Unit)
			}
		}
	})
}

func (w *WatchCommand) updateStatus(gui *gocui.Gui, state *nagioswatcher.StateNotification) {
	_, has := w.status[state.Identity]
	if !has {
		w.status[state.Identity] = map[string]string{}
	}
	w.status[state.Identity][state.Machine] = state.Status

	ok, warn, crit, unknown := 0, 0, 0, 0
	for _, node := range w.status {
		for _, val := range node {
			switch val {
			case "OK":
				ok++
			case "CRITICAL":
				crit++
			case "WARNING":
				warn++
			case "UNKNOWN":
				unknown++
			}
		}
	}
	w.updateView(gui, "Status", false, func(o io.Writer, vw *gocui.View) {
		vw.Clear()

		if crit > 0 {
			vw.FgColor = gocui.ColorRed
		} else if warn > 0 {
			vw.FgColor = gocui.ColorYellow
		} else if unknown > 0 {
			vw.FgColor = gocui.ColorDefault
		} else if ok > 0 {
			vw.FgColor = gocui.ColorGreen
		}

		fmt.Fprintf(o, "\tOK: %d WARNING: %d CRITICAL: %d UNKNOWN: %d", ok, warn, crit, unknown)
	})
}

func (w *WatchCommand) updateView(gui *gocui.Gui, view string, buffered bool, t func(io.Writer, *gocui.View)) {
	gui.Update(func(g *gocui.Gui) error {
		vw, err := g.View(view)
		if err != nil {
			return nil
		}

		if !buffered {
			t(vw, vw)
			return nil
		}

		var buf bytes.Buffer
		t(&buf, vw)

		vb, ok := w.vwBuffers[view]
		if !ok {
			w.vwBuffers[view] = []string{}
		}

		if len(vb) > 300 {
			old := w.vwBuffers[view]
			w.vwBuffers[view] = []string{}
			w.vwBuffers[view] = old[150:]
			vw.Clear()
			for _, line := range w.vwBuffers[view] {
				fmt.Fprint(vw, line)
			}
		}

		line := buf.String()
		w.vwBuffers[view] = append(w.vwBuffers[view], line)
		fmt.Fprint(vw, line)

		return nil
	})
}

func (w *WatchCommand) setupWindows() (gui *gocui.Gui, err error) {
	g, err := gocui.NewGui(gocui.Output256, false)
	if err != nil {
		return nil, err
	}

	offset := 0
	layout := func(g *gocui.Gui) error {
		maxX, maxY := g.Size()
		midY := (maxY / 5) * 4

		// dont make transitions too small
		if midY+offset < 4 {
			w.Lock()
			offset = (midY * -1) + 3
			w.Unlock()
		}

		// dont make status too small
		if midY+offset > maxY-9 {
			w.Lock()
			offset = maxY - 9 - midY
			w.Unlock()
		}

		t, err := g.SetView("Checks", 0, 0, maxX-1, midY+offset, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			t.Autoscroll = true
			t.Overwrite = true
			t.Title = " Checks "
			t.Frame = true
		}

		c, err := g.SetView("Transitions", 0, midY+offset+1, maxX-1, maxY-5, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			c.Autoscroll = true
			c.Overwrite = true
			c.Title = " Transitions "
			c.Frame = true
		}

		s, err := g.SetView("Status", 0, maxY-4, maxX-1, maxY-2, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			s.Frame = true
			s.Title = " Observed Status "
			fmt.Fprintf(s, "Waiting for updates...")
		}

		h, err := g.SetView("Help", 0, maxY-2, maxX-1, maxY, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			h.Frame = false
			idf := ""
			cf := ""
			if w.identity != "" {
				idf = fmt.Sprintf(" identity %q", w.identity)
			}
			if w.check != "" {
				cf = fmt.Sprintf(" check %q", w.check)
			}

			if idf != "" || cf != "" {
				fmt.Fprintf(h, "Choria Scout Event Viewer: showing%s%s. Arrows resize, ^R reset view, ^L clear, ^C to exit", idf, cf)
			} else {
				fmt.Fprintf(h, "Choria Scout Event Viewer showing all events. Arrows resize, ^R reset view, ^L clear, ^C to exit")
			}
		}

		return nil
	}

	g.SetManagerFunc(layout)
	err = g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		w.Lock()
		offset++
		w.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		w.Lock()
		offset--
		w.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		w.Lock()
		offset = 0
		w.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error { return gocui.ErrQuit })
	if err != nil {
		g.Close()
		return nil, err
	}

	err = g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error { return gocui.ErrQuit })
	if err != nil {
		g.Close()
		return nil, err
	}

	err = g.SetKeybinding("", gocui.KeyCtrlL, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		vw, err := g.View("Transitions")
		if err == nil {
			vw.Clear()
		}
		vw, err = g.View("Checks")
		if err == nil {
			vw.Clear()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (w *WatchCommand) subscribeJetStream(ctx context.Context, transitions chan *nats.Msg, states chan *nats.Msg) error {
	mgr, err := jsm.New(w.nc.Nats())
	if err != nil {
		return err
	}

	str, err := mgr.LoadStream("CHORIA_MACHINE")
	if err != nil {
		return err
	}

	le := w.Logrus.(*logrus.Entry)

	w.transEph, err = stream.NewEphemeral(ctx, w.nc.Nats(), str, time.Minute, transitions, le, jsm.FilterStreamBySubject("choria.machine.transition"), jsm.StartAtTimeDelta(w.history), jsm.AcknowledgeExplicit(), jsm.MaxAckPending(50), jsm.MaxDeliveryAttempts(1))
	if err != nil {
		return fmt.Errorf("could not subscribe to Choria Streaming stream CHORIA_MACHINE: %s", err)
	}

	w.stateEph, err = stream.NewEphemeral(ctx, w.nc.Nats(), str, time.Minute, states, le, jsm.FilterStreamBySubject("choria.machine.watcher.nagios.state"), jsm.StartAtTimeDelta(w.history), jsm.AcknowledgeExplicit(), jsm.MaxAckPending(50), jsm.MaxDeliveryAttempts(1))
	if err != nil {
		return fmt.Errorf("could not subscribe to Choria Streaming stream CHORIA_MACHINE: %s", err)
	}

	return nil
}

func (w *WatchCommand) subscribeDirect(transitions chan *nats.Msg, states chan *nats.Msg) error {
	nc := w.nc.Nats()
	_, err := nc.ChanSubscribe("choria.machine.transition", transitions)
	if err != nil {
		return fmt.Errorf("could not subscribe to transitions: %s", err)
	}

	_, err = nc.ChanSubscribe("choria.machine.watcher.nagios.state", states)
	if err != nil {
		return fmt.Errorf("could not subscribe to states: %s", err)
	}

	return nil
}
