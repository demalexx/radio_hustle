package radio_hustle_calendar

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	calendarSummary     = "Хастл турниры"
	calendarDescription = "От Radio Hustle\nhttps://app.radiohustle.online/#/calendar"
)

// Competition as radio-hustle returns it
type Competition struct {
	ForumTopicIndex string `json:"forumTopicIndex"`
	Name            string `json:"compname"`
	City            string `json:"city"`
	Day             Day    `json:"date"`
}

// Competition and Google Calendar Event together.
// Competition and Event are optional,
// but at least one should be presented
type CompetitionAndEvent struct {
	Competition     *Competition
	Event           *calendar.Event
	CalendarEventId string

	calendarService *calendar.Service
	calendarId      string
}

type Syncer struct {
	Data []*CompetitionAndEvent

	calendarService *calendar.Service
	calendarId      string
}

func CreateSyncer(
	service *calendar.Service,
	calendarId string,
) Syncer {
	syncer := Syncer{
		calendarService: service,
		calendarId:      calendarId,
	}

	cal, err := service.Calendars.Get(calendarId).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(
		"Working with calendar %s %s",
		cal.Summary,
		cal.Id,
	)
	log.Printf(
		"Calendar link: https://calendar.google.com/calendar?cid=%s",
		base64.StdEncoding.EncodeToString([]byte(cal.Id)),
	)

	calendarEvents := syncer.getCalendarEvents()
	competitions := syncer.getCompetitions()

	syncer.prepareSync(calendarEvents, competitions)

	return syncer
}

func (t *Syncer) Sync() SyncResult {
	t.syncCalendar()

	syncResult := SyncResult{}
	for i := range t.Data {
		code := t.Data[i].Sync()
		if code == -1 {
			syncResult.Deleted += 1
		} else if code == 0 {
			syncResult.UpToDate += 1
		} else if code == 1 {
			syncResult.Added += 1
		} else if code == 2 {
			syncResult.Updated += 1
		}
	}
	return syncResult
}

func (t *Syncer) prepareSync(
	calendarEvents []calendar.Event,
	competitions []Competition,
) {
	var calendarEventsById = map[string]*calendar.Event{}
	for i := range calendarEvents {
		calendarEventsById[calendarEvents[i].Id] = &calendarEvents[i]
	}

	for i := range competitions {
		competitionAndEvent := CompetitionAndEvent{
			Competition:     &competitions[i],
			CalendarEventId: fmt.Sprintf("000%s", competitions[i].ForumTopicIndex),
			calendarService: t.calendarService,
			calendarId:      t.calendarId,
		}

		event, ok := calendarEventsById[competitionAndEvent.CalendarEventId]
		if ok == true {
			competitionAndEvent.Event = event
		}
		delete(calendarEventsById, competitionAndEvent.CalendarEventId)

		t.Data = append(t.Data, &competitionAndEvent)
	}

	for i := range calendarEventsById {
		t.Data = append(t.Data, &CompetitionAndEvent{
			Event:           calendarEventsById[i],
			CalendarEventId: i,
			calendarService: t.calendarService,
			calendarId:      t.calendarId,
		})
	}
}

func (t *Syncer) getCalendarEvents() []calendar.Event {
	log.Printf("Getting calendar events")
	var result []calendar.Event
	var pageToken string
	for {
		events, err := t.calendarService.Events.List(
			t.calendarId,
		).PageToken(pageToken).Do()
		if err != nil {
			log.Fatal(err)
		}

		for i := range events.Items {
			result = append(result, *events.Items[i])
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			break
		}
	}

	log.Printf("Received calendar events: %d", len(result))

	return result
}

func (t *Syncer) getCompetitions() []Competition {
	// Get competitions from radio-hustle
	const competitionsUrl = "https://data.radiohustle.online/db/getCompetitions/"

	log.Printf("Getting competitions from %s", competitionsUrl)
	response, err := http.Get(competitionsUrl)
	if err != nil {
		log.Fatal(err)
	}
	bodyJson, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result []Competition
	err = json.Unmarshal(bodyJson, &result)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Received competitions: %d", len(result))

	return result
}

func (t *Syncer) syncCalendar() {
	cal, err := t.calendarService.Calendars.Get(t.calendarId).Do()
	if err != nil {
		log.Fatal(err)
	}
	cal.Summary = calendarSummary
	cal.Description = calendarDescription
	_, err = t.calendarService.Calendars.Update(t.calendarId, cal).Do()
	if err != nil {
		log.Fatal(err)
	}
}

func (t *CompetitionAndEvent) Sync() int {
	if t.Event == nil {
		t.addCalendarEvent()
		return 1
	} else if t.Competition == nil {
		t.delCalendarEvent()
		return -1
	} else {
		if t.needUpdateCalendarEvent() {
			t.updateCalendarEvent()
			return 2
		} else {
			return 0
		}
	}
}

func (t *CompetitionAndEvent) needUpdateCalendarEvent() bool {
	expectedEvent := t.getExpectedCalendarEvent()

	return expectedEvent.Start.Date != t.Event.Start.Date ||
		expectedEvent.End.Date != t.Event.End.Date ||
		expectedEvent.Summary != t.Event.Summary ||
		expectedEvent.Description != t.Event.Description ||
		expectedEvent.Location != t.Event.Location
}

func (t *CompetitionAndEvent) getExpectedCalendarEvent() *calendar.Event {
	day := t.Competition.Day.Format(time.DateOnly)
	return &calendar.Event{
		Start:   &calendar.EventDateTime{Date: day},
		End:     &calendar.EventDateTime{Date: day},
		Summary: t.Competition.Name,
		Description: fmt.Sprintf(
			"http://hustle-sa.ru/forum/index.php?showtopic=%s",
			t.Competition.ForumTopicIndex,
		),
		Location: t.Competition.City,
	}
}

func (t *CompetitionAndEvent) addCalendarEvent() {
	expectedEvent := t.getExpectedCalendarEvent()
	expectedEvent.Id = t.CalendarEventId

	event, err := t.calendarService.Events.Insert(
		t.calendarId,
		expectedEvent,
	).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Event created: %s %s %s", event.Id, event.Start.Date, event.Summary)
}

func (t *CompetitionAndEvent) updateCalendarEvent() {
	event, err := t.calendarService.Events.Update(
		t.calendarId,
		t.CalendarEventId,
		t.getExpectedCalendarEvent(),
	).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Event updated: %s %s %s", event.Id, event.Start.Date, event.Summary)
}

func (t *CompetitionAndEvent) delCalendarEvent() {
	err := t.calendarService.Events.Delete(t.calendarId, t.Event.Id).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(
		"Event deleted: %s %s %s",
		t.Event.Id,
		t.Event.Start.Date,
		t.Event.Summary,
	)
}

func GetCalendarService(credentialsFile string) *calendar.Service {
	log.Printf("Getting Google Calendar Service")
	result, err := calendar.NewService(
		context.Background(),
		option.WithCredentialsFile(credentialsFile),
	)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func ListCalendars(srv *calendar.Service) {
	calendars, err := srv.CalendarList.List().Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Calendars:\n")
	for _, calendarItem := range calendars.Items {
		log.Printf("ID: %s", calendarItem.Id)
		log.Printf("  Summary: %s", calendarItem.Summary)

		events, err := srv.Events.List(calendarItem.Id).Do()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("  Updated: %s", events.Updated)

		acl, err := srv.Acl.List(calendarItem.Id).Do()
		if err != nil {
			log.Fatal(err)
		}
		for _, aclItem := range acl.Items {
			log.Printf("  %s %s", aclItem.Role, aclItem.Id)
		}
	}
}

func DelCalendar(srv *calendar.Service, calendarId string) {
	err := srv.Calendars.Delete(calendarId).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Calendar deleted, ID: %s", calendarId)
}

func CreateCalendar(srv *calendar.Service) {
	calendarNew, err := srv.Calendars.Insert(&calendar.Calendar{
		Summary:     calendarSummary,
		Description: calendarDescription,
	}).Do()
	if err != nil {
		panic(err)
	}

	_, err = srv.Acl.Insert(
		calendarNew.Id,
		&calendar.AclRule{
			Role: "reader",
			Scope: &calendar.AclRuleScope{
				Type: "default",
			},
		},
	).SendNotifications(false).Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"Calendar created, made publicly visible, ID: %s",
		calendarNew.Id,
	)
}
