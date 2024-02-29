package radio_hustle_calendar

import (
	"log"
	"time"
)

type Day struct {
	time.Time
}

func (t *Day) UnmarshalJSON(b []byte) (err error) {
	dayTime, err := time.Parse(`"02.01.2006"`, string(b))
	if err != nil {
		log.Fatal(err)
	}
	t.Time = dayTime
	return
}

type SyncResult struct {
	Added    int
	Deleted  int
	Updated  int
	UpToDate int
}
