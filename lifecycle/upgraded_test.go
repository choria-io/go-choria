// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UpgradedEvent", func() {
	Describe("newUpgradedEvent", func() {
		It("Should create the event and set options", func() {
			event := newUpgradeEvent(Component("ginkgo"))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Upgraded))
		})
	})

	Describe("newUpgradedEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newUpgradeEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newUpgradeEventFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.upgraded", "component":"ginkgo", "version":"1.2.3","new_version":"1.2.4"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Upgraded))
			Expect(event.TypeString()).To(Equal("upgraded"))
			Expect(event.Version).To(Equal("1.2.3"))
			Expect(event.NewVersion).To(Equal("1.2.4"))
		})
	})

	Describe("SetVersion", func() {
		It("Set the version", func() {
			e := &UpgradedEvent{}
			e.SetVersion("1.2.3")
			Expect(e.Version).To(Equal("1.2.3"))
		})
	})

	Describe("SetNewVersion", func() {
		It("Set the new version", func() {
			e := &UpgradedEvent{}
			e.SetNewVersion("1.2.3")
			Expect(e.NewVersion).To(Equal("1.2.3"))
		})
	})

	Describe("Target", func() {
		It("Should detect incomplete events", func() {
			e := &UpgradedEvent{}
			_, err := e.Target()
			Expect(err).To(MatchError("event is not complete, component has not been set"))
		})

		It("Should return the right target", func() {
			e := newUpgradeEvent(Component("ginkgo"))
			t, err := e.Target()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal("choria.lifecycle.event.upgraded.ginkgo"))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := newUpgradeEvent(Component("ginkgo"), Identity("node.example"), Version("1.2.3"), NewVersion("1.2.4"))
			Expect(e.String()).To(Equal("[upgraded] node.example: ginkgo version 1.2.3 to 1.2.4"))
		})
	})
})
