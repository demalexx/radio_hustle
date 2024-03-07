package radio_hustle_calendar

import (
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"testing"
	"time"
)

func TestGetCompetitions(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"GET",
		"https://data.radiohustle.online/db/getCompetitions/",
		httpmock.NewStringResponder(
			200,
			`[{"forumTopicIndex":"8947","compname":"Открытие сезона","city":"Москва","date":"29.08.2024"}]`,
		),
	)

	actual := (&Syncer{}).getCompetitions()
	expected := []Competition{
		{
			ForumTopicIndex: "8947",
			Name:            "Открытие сезона",
			City:            "Москва",
			Day:             Day{time.Date(2024, 8, 29, 0, 0, 0, 0, time.UTC)},
		},
	}
	require.Equal(t, expected, actual)
}

func TestPrepareSync(t *testing.T) {
	competitions := []Competition{
		{
			ForumTopicIndex: "1",
			Name:            "comp-1",
			City:            "City-1",
			Day:             Day{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	events := []calendar.Event{
		{
			Id: "0001",
		},
		{
			Id: "event-with-no-competition",
		},
	}

	syncer := &Syncer{}
	syncer.prepareSync(events, competitions)
	actual := syncer.Data

	require.Equal(t, 2, len(actual))

	// 1st element contains matched radio hustle competition
	// and calendar event
	expected := &CompetitionAndEvent{
		Competition:     &competitions[0],
		Event:           &events[0],
		CalendarEventId: "0001",
	}
	require.Equal(t, *expected, *actual[0])

	// 2nd element contains just calendar event
	expected = &CompetitionAndEvent{
		Competition:     nil,
		Event:           &events[1],
		CalendarEventId: "event-with-no-competition",
	}
	require.Equal(t, *expected, *actual[1])
}

func TestGetExpectedCalendarEvent(t *testing.T) {
	competitionAndEvent := &CompetitionAndEvent{
		Competition: &Competition{
			ForumTopicIndex: "1",
			Name:            "comp-1",
			City:            "city-1",
			Day:             Day{time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
		},
	}
	actual := competitionAndEvent.getExpectedCalendarEvent()
	expected := &calendar.Event{
		Start:       &calendar.EventDateTime{Date: "2024-02-03"},
		End:         &calendar.EventDateTime{Date: "2024-02-03"},
		Summary:     "comp-1",
		Description: "http://hustle-sa.ru/forum/index.php?showtopic=1",
		Location:    "city-1",
	}

	require.Equal(t, expected, actual)
}

func TestNeedUpdateCalendarEvent(t *testing.T) {
	type test struct {
		eventSourceFn func() *calendar.Event
		want          bool
	}

	eventNoNeedUpdate := calendar.Event{
		Start:       &calendar.EventDateTime{Date: "2024-02-03"},
		End:         &calendar.EventDateTime{Date: "2024-02-03"},
		Summary:     "comp 1",
		Description: "http://hustle-sa.ru/forum/index.php?showtopic=1",
		Location:    "city 1",
	}

	tests := []test{
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				return &result
			},
			want: false,
		},
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				result.Start = &calendar.EventDateTime{Date: "2024-01-01"}
				return &result
			},
			want: true,
		},
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				result.End = &calendar.EventDateTime{Date: "2024-01-01"}
				return &result
			},
			want: true,
		},
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				result.Summary = "need update"
				return &result
			},
			want: true,
		},
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				result.Description = "need update"
				return &result
			},
			want: true,
		},
		{
			eventSourceFn: func() *calendar.Event {
				result := eventNoNeedUpdate
				result.Location = "need update"
				return &result
			},
			want: true,
		},
	}

	for _, testCase := range tests {
		t.Run("", func(t *testing.T) {
			competitionAndEvent := &CompetitionAndEvent{
				Competition: &Competition{
					ForumTopicIndex: "1",
					Name:            "comp 1",
					City:            "city 1",
					Day:             Day{time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)},
				},
				Event: testCase.eventSourceFn(),
			}
			actual := competitionAndEvent.needUpdateCalendarEvent()
			require.Equal(t, testCase.want, actual)
		})
	}
}
