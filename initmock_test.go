package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtract(t *testing.T) {
	Convey("extract skip", t, func() {
		Convey("1", func() {
			argmap, rest := extractArgs([]string{"-a", "-b", "-skippkg", "abc"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"-a", "-b"})
		})
		Convey("2", func() {
			argmap, rest := extractArgs([]string{"-skippkg", "abc", "-a", "-b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"-a", "-b"})
		})
		Convey("3", func() {
			argmap, rest := extractArgs([]string{"-skippkg", "abc"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"abc"})
			So(rest, ShouldBeEmpty)
		})
		Convey("4", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg=abc"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("5", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skippkg"})
		})
		Convey("6", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg="}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skippkg="})
		})
		Convey("7", func() {
			argmap, rest := extractArgs([]string{"a", "--skippkg"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "--skippkg"})
		})
		Convey("8", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg=a", "-skippkg=b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"a", "b"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("9", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg=a", "-skippkg", "b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"a", "b"})
			So(rest, ShouldResemble, []string{"a"})
		})
		Convey("10", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg", "abc", "b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldResemble, []string{"abc"})
			So(rest, ShouldResemble, []string{"a", "b"})
		})
		Convey("11", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg=", "abc", "b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skippkg=", "abc", "b"})
		})
		Convey("12", func() {
			argmap, rest := extractArgs([]string{"a", "-skippkg=-skippkg=", "abc", "b"}, []string{SkipPkgFlag})
			So(argmap[SkipPkgFlag], ShouldBeEmpty)
			So(rest, ShouldResemble, []string{"a", "-skippkg=-skippkg=", "abc", "b"})
		})
	})
}
