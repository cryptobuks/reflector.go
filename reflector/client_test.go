package reflector

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/lbryio/reflector.go/store"
)

var address = "localhost:" + strconv.Itoa(DefaultPort)
var s Server

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "reflector_client_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	ms := store.MemoryBlobStore{}
	s := NewServer(&ms)
	go s.ListenAndServe(address)

	os.Exit(m.Run())
}

func TestNotConnected(t *testing.T) {
	c := Client{}
	err := c.SendBlob([]byte{})
	if err == nil {
		t.Error("client should error if it is not connected")
	}
}

func TestSmallBlob(t *testing.T) {
	c := Client{}
	err := c.Connect(address)
	if err != nil {
		t.Error(err)
	}

	err = c.SendBlob([]byte{})
	if err == nil {
		t.Error("client should error if blob is empty")
	}

	blob := make([]byte, 1000)
	_, err = rand.Read(blob)
	if err != nil {
		t.Error("failed to make random blob")
	}

	err = c.SendBlob([]byte{})
	if err == nil {
		t.Error("client should error if blob is the wrong size")
	}
}
