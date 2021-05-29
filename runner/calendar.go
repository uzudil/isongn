package runner

import "fmt"

type CalendarUpdate interface {
	MinsChange(mins, hours, day, month, year int)
}

type Calendar struct {
	Paused          bool
	MinsSinceEpoch  int
	incSpeed, ticks float64
	EventListener   CalendarUpdate
}

func NewCalendar(mins, hours, day, month, year int, incSpeed float64) *Calendar {
	return &Calendar{false, ToEpoch(mins, hours, day, month, year), incSpeed, 0, nil}
}

func (c *Calendar) Incr(delta float64) {
	if c.Paused {
		return
	}
	c.ticks += delta
	if c.ticks >= c.incSpeed {
		c.ticks = 0
		c.MinsSinceEpoch++
		if c.EventListener != nil {
			c.EventListener.MinsChange(c.FromEpoch())
		}
	}
}

const (
	ticksPerHour  = 60
	ticksPerDay   = ticksPerHour * 24
	ticksPerMonth = ticksPerDay * 30
	ticksPerYear  = ticksPerMonth * 12
)

func (c *Calendar) FromEpoch() (mins, hours, day, month, year int) {
	year = c.MinsSinceEpoch / ticksPerYear
	month = (c.MinsSinceEpoch % ticksPerYear) / ticksPerMonth
	day = ((c.MinsSinceEpoch % ticksPerYear) % ticksPerMonth) / ticksPerDay
	hours = (((c.MinsSinceEpoch % ticksPerYear) % ticksPerMonth) % ticksPerDay) / ticksPerHour
	mins = (((c.MinsSinceEpoch % ticksPerYear) % ticksPerMonth) % ticksPerDay) % ticksPerHour
	return
}

func (c *Calendar) AsString() string {
	mins, hours, day, month, year := c.FromEpoch()
	return fmt.Sprintf("%04d/%02d/%02d %02d:%02d", year, month, day, hours, mins)
}

func (c *Calendar) AsTimeString() string {
	mins, hours, _, _, _ := c.FromEpoch()
	return fmt.Sprintf("%02d:%02d", hours, mins)
}

func ToEpoch(mins, hours, day, month, year int) int {
	return year*ticksPerYear + month*ticksPerMonth + day*ticksPerDay + hours*ticksPerHour + mins
}
