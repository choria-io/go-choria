// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InputItem", func() {
	Describe("ValidateInputValue", func() {
		var input InputItem
		BeforeEach(func() {
			input = InputItem{}
		})

		It("Should validate integer", func() {
			input.Type = InputTypeInteger
			warnings, err := input.ValidateValue(10)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("a")
			Expect(err).To(MatchError("is not an integer"))
			Expect(warnings).To(BeEmpty())

			i := map[string]any{}
			err = json.Unmarshal([]byte(`{"x":1}`), &i)
			Expect(err).ToNot(HaveOccurred())
			warnings, err = input.ValidateValue(i["x"])
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			err = json.Unmarshal([]byte(`{"x":1.1}`), &i)
			Expect(err).ToNot(HaveOccurred())
			warnings, err = input.ValidateValue(i["x"])
			Expect(err).To(MatchError("is not an integer"))
			Expect(warnings).To(BeEmpty())

			converted, warnings, err := input.ValidateStringValue("10")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())
			Expect(converted).To(BeAssignableToTypeOf(int64(1)))
		})

		It("Should validate number", func() {
			input.Type = InputTypeNumber
			warnings, err := input.ValidateValue(10)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue(10.2)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("a")
			Expect(err).To(MatchError("is not a number"))
			Expect(warnings).To(BeEmpty())
		})

		It("Should validate float", func() {
			input.Type = InputTypeFloat
			warnings, err := input.ValidateValue(10.2)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			v, err := input.ConvertStringValue("200.0")
			Expect(err).ToNot(HaveOccurred())
			warnings, err = input.ValidateValue(v)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("a")
			Expect(err).To(MatchError("is not a float"))
			Expect(warnings).To(BeEmpty())
		})

		It("Should validate string", func() {
			input.Type = InputTypeString
			input.MaxLength = 20
			warnings, err := input.ValidateValue("hello world")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("123456789012345678901234567890")
			Expect(err).To(MatchError("is longer than 20 characters"))
			Expect(warnings).To(BeEmpty())

			input.MaxLength = 0
			warnings, err = input.ValidateValue("123456789012345678901234567890")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should validate boolean", func() {
			input.Type = InputTypeBoolean
			warnings, err := input.ValidateValue(true)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue(false)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("foo")
			Expect(err).To(MatchError("is not a boolean"))
			Expect(warnings).To(BeEmpty())
		})

		It("Should validate list", func() {
			input.Type = InputTypeList
			input.Enum = []string{"one", "two"}
			warnings, err := input.ValidateValue("one")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			warnings, err = input.ValidateValue("two")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			_, err = input.ValidateValue("three")
			Expect(err).To(MatchError("should be one of one, two"))

			_, err = input.ValidateValue(false)
			Expect(err).To(MatchError("should be a string"))
		})

		It("Should validate unknowns", func() {
			input.Type = "unknown"

			warnings, err := input.ValidateValue("one")
			Expect(warnings).To(BeEmpty())
			Expect(err).To(MatchError("unsupported input type 'unknown'"))
		})

		It("Should validate string content", func() {
			input.Type = InputTypeString
			input.Validation = "^bob"
			input.MaxLength = 20
			warnings, err := input.ValidateValue("hello world")
			Expect(err).To(MatchError("input does not match '^bob'"))
			Expect(warnings).To(BeEmpty())

			input.Validation = "bob"
			warnings, err = input.ValidateValue("hello world")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(Equal([]string{"Unsupported validator 'bob'"}))
		})

		It("Should validate hash content", func() {
			input.Type = InputTypeHash
			warnings, err := input.ValidateValue("hello world")
			Expect(warnings).To(BeEmpty())
			Expect(err).To(MatchError("is not a hash map"))

			warnings, err = input.ValidateValue(map[string]string{"hello": "world"})
			Expect(warnings).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should validate array content", func() {
			input.Type = InputTypeArray
			warnings, err := input.ValidateValue("hello world")
			Expect(warnings).To(BeEmpty())
			Expect(err).To(MatchError("is not an array"))

			warnings, err = input.ValidateValue([]string{"hello", "world"})
			Expect(warnings).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
