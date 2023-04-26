// Copyright (c) 2021-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package pluginswatcher

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/ghodss/yaml"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/PluginsWatcher")
}

var _ = Describe("AAgent/Watchers/PluginsWatcher", func() {
	var (
		w       *Watcher
		machine *model.MockMachine
		mockctl *gomock.Controller
		td      string
		err     error
	)

	BeforeEach(func() {
		td, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		mockctl = gomock.NewController(GinkgoT())

		machine = model.NewMockMachine(mockctl)
		machine.EXPECT().Directory().Return(td).AnyTimes()
		machine.EXPECT().SignerKey().Return("").AnyTimes()

		wi, err := New(machine, "machines", nil, "", "", "1m", time.Hour, map[string]any{
			"data_item": "spec",
		})
		Expect(err).ToNot(HaveOccurred())
		w = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
		os.RemoveAll(td)
	})

	Describe("Specification/Encode", func() {
		It("Should correctly encode the specification", func() {
			pub, priv, err := iu.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			data, err := os.ReadFile("testdata/plugins.json")
			Expect(err).ToNot(HaveOccurred())

			spec := &Specification{Plugins: data}
			_, err = spec.Encode(hex.EncodeToString(priv))
			Expect(err).ToNot(HaveOccurred())

			sig, err := hex.DecodeString(spec.Signature)
			Expect(err).ToNot(HaveOccurred())

			ok, err := iu.Ed25519Verify(pub, data, sig)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})

	Describe("setProperties", func() {
		It("Should support defaulting manager machine prefix", func() {
			err = w.setProperties(map[string]any{})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.properties.ManagerMachinePrefix).To(Equal("mm"))
		})

		It("Should not allow underscores", func() {
			err = w.setProperties(map[string]any{"manager_machine_prefix": "plugin_foo"})
			Expect(err).To(MatchError("manager_machine_prefix may not contain underscore"))
		})

		It("Should support custom manager machine prefix", func() {
			err = w.setProperties(map[string]any{"manager_machine_prefix": "plugin"})
			Expect(err).ToNot(HaveOccurred())

			Expect(w.properties.ManagerMachinePrefix).To(Equal("plugin"))
			r, err := w.renderMachine(&ManagedPlugin{
				Name:       "x",
				NamePrefix: "plugin",
			})
			Expect(err).ToNot(HaveOccurred())

			var m map[string]any
			Expect(yaml.Unmarshal(r, &m)).To(Succeed())

			Expect(m["name"]).To(Equal("plugin_x"))
			actual := w.targetDirForManagerMachine("x")
			expected := filepath.Join(filepath.Dir(w.machine.Directory()), "plugin_x")
			Expect(actual).To(Equal(expected))
		})
	})

	Describe("targetDirForManagerMachine", func() {
		It("Should set the correct target", func() {
			err = w.setProperties(map[string]any{"manager_machine_prefix": "plugin"})
			Expect(err).ToNot(HaveOccurred())
			actual := w.targetDirForManagerMachine("x")
			expected := filepath.Join(filepath.Dir(w.machine.Directory()), "plugin_x")
			Expect(actual).To(Equal(expected))
		})
	})
	Describe("loadAndValidateData", func() {
		var (
			data *Specification
			pri  ed25519.PrivateKey
			pub  ed25519.PublicKey
			spec []byte
		)

		BeforeEach(func() {
			pub, pri, err = ed25519.GenerateKey(rand.Reader)
			Expect(err).ToNot(HaveOccurred())
			spec = []byte("[]")
			data = &Specification{Plugins: spec}
			data.Signature = hex.EncodeToString(ed25519.Sign(pri, spec))
			machine.EXPECT().DataGet(gomock.Eq("spec")).Return(data, true).AnyTimes()
		})

		It("Should function without a signature", func() {
			data.Signature = ""
			machine.EXPECT().SignerKey().Return("").AnyTimes()
			spec, err := w.loadAndValidateData()
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).ToNot(BeNil())
		})

		It("Should handle data with no signatures when signature is required", func() {
			err = w.setProperties(map[string]any{
				"data_item":  "spec",
				"public_key": "x",
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = ""
			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("No signature found in specification, removing data"))
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should handle data with corrupt signatures", func() {
			err = w.setProperties(map[string]any{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = "x"

			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("Signature in data_item %s did not verify using configured public key '%s', removing data"), gomock.Eq("spec"), gomock.Eq(hex.EncodeToString(pub)))
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should handle data with invalid signatures", func() {
			err = w.setProperties(map[string]any{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = hex.EncodeToString(ed25519.Sign(pri, []byte("wrong")))

			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("Signature in data_item %s did not verify using configured public key '%s', removing data"), gomock.Eq("spec"), gomock.Eq(hex.EncodeToString(pub)))
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should allow overrides from config", func() {
			machine := model.NewMockMachine(mockctl)
			machine.EXPECT().Directory().Return(td).AnyTimes()
			machine.EXPECT().SignerKey().Return(hex.EncodeToString(pub)).AnyTimes()
			machine.EXPECT().DataGet(gomock.Eq("spec")).Return(data, true).AnyTimes()

			wi, err := New(machine, "machines", nil, "", "", "1m", time.Hour, map[string]any{
				"data_item":  "spec",
				"public_key": "other",
			})
			Expect(err).ToNot(HaveOccurred())
			w = wi.(*Watcher)

			Expect(w.properties.PublicKey).To(Equal(hex.EncodeToString(pub)))

			spec, err := w.loadAndValidateData()
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(Equal([]byte("[]")))
		})

		It("Should handle valid signatures", func() {
			err = w.setProperties(map[string]any{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())

			spec, err := w.loadAndValidateData()
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(Equal([]byte("[]")))
		})
	})
})
