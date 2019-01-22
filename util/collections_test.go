package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchesAny(t *testing.T) {
	t.Parallel()

	realWorldErrorMessages := []string{
		"Failed to load state: RequestError: send request failed\ncaused by: Get https://<BUCKET_NAME>.us-west-2.amazonaws.com/?prefix=env%3A%2F: dial tcp 54.231.176.160:443: i/o timeout",
		"aws_cloudwatch_metric_alarm.asg_high_memory_utilization: Creating metric alarm failed: ValidationError: A separate request to update this alarm is in progress. status code: 400, request id: 94309fbd-7e09-11e8-a5f8-1de9e697c6bf",
		"Error configuring the backend \"s3\": RequestError: send request failed\ncaused by: Post https://sts.amazonaws.com/: net/http: TLS handshake timeout",
	}

	testCases := []struct {
		list     []string
		element  string
		expected bool
	}{
		{nil, "", false},
		{[]string{}, "", false},
		{[]string{}, "foo", false},
		{[]string{"foo"}, "kafoot", true},
		{[]string{"bar", "foo", ".*Failed to load backend.*TLS handshake timeout.*"}, "Failed to load backend: Error...:...  TLS handshake timeout", true},
		{[]string{"bar", "foo", ".*Failed to load backend.*TLS handshake timeout.*"}, "Failed to load backend: Error...:...  TLxS handshake timeout", false},
		{[]string{"(?s).*Failed to load state.*dial tcp.*timeout.*"}, realWorldErrorMessages[0], true},
		{[]string{"(?s).*Creating metric alarm failed.*request to update this alarm is in progress.*"}, realWorldErrorMessages[1], true},
		{[]string{"(?s).*Error configuring the backend.*TLS handshake timeout.*"}, realWorldErrorMessages[2], true},
	}

	for _, testCase := range testCases {
		actual := MatchesAny(testCase.list, testCase.element)
		assert.Equal(t, testCase.expected, actual, "For list %v and element %s", testCase.list, testCase.element)
	}
}

func TestListContainsElement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		list     []string
		element  string
		expected bool
	}{
		{[]string{}, "", false},
		{[]string{}, "foo", false},
		{[]string{"foo"}, "foo", true},
		{[]string{"bar", "foo", "baz"}, "foo", true},
		{[]string{"bar", "foo", "baz"}, "nope", false},
		{[]string{"bar", "foo", "baz"}, "", false},
	}

	for _, testCase := range testCases {
		actual := ListContainsElement(testCase.list, testCase.element)
		assert.Equal(t, testCase.expected, actual, "For list %v and element %s", testCase.list, testCase.element)
	}
}

func TestRemoveElementFromList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		list     []string
		element  string
		expected []string
	}{
		{[]string{}, "", []string{}},
		{[]string{}, "foo", []string{}},
		{[]string{"foo"}, "foo", []string{}},
		{[]string{"bar"}, "foo", []string{"bar"}},
		{[]string{"bar", "foo", "baz"}, "foo", []string{"bar", "baz"}},
		{[]string{"bar", "foo", "baz"}, "nope", []string{"bar", "foo", "baz"}},
		{[]string{"bar", "foo", "baz"}, "", []string{"bar", "foo", "baz"}},
	}

	for _, testCase := range testCases {
		actual := RemoveElementFromList(testCase.list, testCase.element)
		assert.Equal(t, testCase.expected, actual, "For list %v and element %s", testCase.list, testCase.element)
	}
}

func TestRemoveDuplicatesFromList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		list     []string
		expected []string
		reverse  bool
	}{
		{[]string{}, []string{}, false},
		{[]string{"foo"}, []string{"foo"}, false},
		{[]string{"foo", "bar"}, []string{"foo", "bar"}, false},
		{[]string{"foo", "bar", "foobar", "bar", "foo"}, []string{"foo", "bar", "foobar"}, false},
		{[]string{"foo", "bar", "foobar", "foo", "bar"}, []string{"foo", "bar", "foobar"}, false},
		{[]string{"foo", "bar", "foobar", "bar", "foo"}, []string{"foobar", "bar", "foo"}, true},
		{[]string{"foo", "bar", "foobar", "foo", "bar"}, []string{"foobar", "foo", "bar"}, true},
	}

	for _, testCase := range testCases {
		f := RemoveDuplicatesFromList
		if testCase.reverse {
			f = RemoveDuplicatesFromListKeepLast
		}
		assert.Equal(t, f(testCase.list), testCase.expected, "For list %v", testCase.list)
		t.Logf("%v passed", testCase.list)
	}
}

func TestCommaSeparatedStrings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		list     []string
		expected string
	}{
		{[]string{}, ``},
		{[]string{"foo"}, `"foo"`},
		{[]string{"foo", "bar"}, `"foo", "bar"`},
	}

	for _, testCase := range testCases {
		assert.Equal(t, CommaSeparatedStrings(testCase.list), testCase.expected, "For list %v", testCase.list)
		t.Logf("%v passed", testCase.list)
	}
}
