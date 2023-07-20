package pkg

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dstotijn/go-notion"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	currentTime := notion.NewDateTime(time.Now(), true)

	spew.Dump(currentTime)
}
