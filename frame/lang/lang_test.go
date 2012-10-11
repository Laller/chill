package lang_test

import(
	"testing"
	"net/url"
	"github.com/opesun/chill/frame/lang"
)

func TestRoute(t *testing.T) {
	path := "/cars/comments"
	query := url.Values{}
	query.Add("make", "bmw")
	query.Add("engine", "4000")
	query.Add("1public", "true")
	route, err := lang.NewRoute(path, query)
	if err != nil {
		t.Fatal()
	}
	if len(route.Words) != 2 {
		t.Fatal()
	}
	if route.Words[0] != "cars" || route.Words[1] != "comments" {
		t.Fatal()
	}
	if route.Queries[0]["make"] == nil || route.Queries[0]["engine"] == nil || route.Queries[1]["public"] == nil {
		t.Fatal()
	}
}

func TestRoute1(t *testing.T) {
	path := "/x/y/z"
	query := url.Values{}
	route, err := lang.NewRoute(path, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(route.Queries) != 3 {
		t.Fatal()
	}
}

func TestRoute2(t *testing.T) {
	path := "/x"
	query := url.Values{}
	query.Add("4hello", "this should fail")
	_, err := lang.NewRoute(path, query)
	if err == nil {
		t.Fatal()
	}
}

type MockSpeaker struct {}

func (m MockSpeaker) IsNoun(s string) bool {
	if s == "cars" || s == "comments" {
		return true
	}
	return false
}

func (m MockSpeaker) NounHasVerb(n, v string) bool {
	if n == "cars" && v == "Ignite" {
		return true
	}
	if n == "comments" && v == "Flame" {
		return true
	}
	return false
}

func TestSentence(t *testing.T) {
	path := "/cars/ignite"
	query := url.Values{}
	route, err := lang.NewRoute(path, query)
	if err != nil {
		t.Fatal()
	}
	speaker := MockSpeaker{}
	sentence, err := lang.NewSentence(route, speaker)
	if err != nil {
		t.Fatal(err)
	}
	if sentence.Noun != "cars" || sentence.Verb != "Ignite" || sentence.Redundant != "" {
		t.Fatal()
	}
}

func TestSentece1(t *testing.T) {
	path := "/cars/not-existing-verb"
	query := url.Values{}
	route, err := lang.NewRoute(path, query)
	if err != nil {
		t.Fatal()
	}
	speaker := MockSpeaker{}
	_, err = lang.NewSentence(route, speaker)
	if err == nil {
		t.Fatal()
	}
}

func TestSentece2(t *testing.T) {
	path := "/not-existing-noun/ignite"
	query := url.Values{}
	route, err := lang.NewRoute(path, query)
	if err != nil {
		t.Fatal()
	}
	speaker := MockSpeaker{}
	_, err = lang.NewSentence(route, speaker)
	if err == nil {
		t.Fatal()
	}
}