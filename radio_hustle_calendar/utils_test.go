package radio_hustle_calendar

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDayUnmarshal(t *testing.T) {
	type testStruct struct {
		Day Day
	}
	var actual []testStruct
	err := json.Unmarshal([]byte(`[{"Day": "01.02.2024"}]`), &actual)

	require.NoError(t, err)

	expected := []testStruct{
		{Day: Day{time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)}},
	}
	require.Equal(t, expected, actual)
}
