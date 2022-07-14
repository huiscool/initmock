package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtract(t *testing.T) {
	Convey("extract skip", t, func() {
		Convey("1", func() {
			skipped, _, rest := extractArgs([]string{"-a", "-b", "-skipinit", "abc"})
			So(skipped, ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"-a", "-b"})
		})
		Convey("2", func() {
			skipped, _, rest := extractArgs([]string{"-skipinit", "abc", "-a", "-b"})
			So(skipped, ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"-a", "-b"})
		})
		Convey("3", func() {
			skipped, _, rest := extractArgs([]string{"-skipinit", "abc"})
			So(skipped, ShouldResemble, []string{"abc"})
			So(rest, ShouldBeEmpty)
		})
		Convey("4", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit=abc"})
			So(skipped, ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("5", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit"})
			So(skipped, ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skipinit"})
		})
		Convey("6", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit="})
			So(skipped, ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skipinit="})
		})
		Convey("7", func() {
			skipped, _, rest := extractArgs([]string{"a", "--skipinit"})
			So(skipped, ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "--skipinit"})
		})
		Convey("8", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit=a", "-skipinit=b"})
			So(skipped, ShouldResemble, []string{"a", "b"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("9", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit=a", "-skipinit", "b"})
			So(skipped, ShouldResemble, []string{"a", "b"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("10", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit", "abc", "b"})
			So(skipped, ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"a", "b"})
		})
		Convey("11", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit=", "abc", "b"})
			So(skipped, ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skipinit=", "abc", "b"})
		})
		Convey("12", func() {
			skipped, _, rest := extractArgs([]string{"a", "-skipinit=-skipinit=", "abc", "b"})
			So(skipped, ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skipinit=-skipinit=", "abc", "b"})
		})
	})
}
