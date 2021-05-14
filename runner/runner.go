package runner

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strconv"

	"github.com/uzudil/bscript/bscript"
	"github.com/uzudil/isongn/gfx"
	"github.com/uzudil/isongn/util"
	"github.com/uzudil/isongn/world"
)

type Message struct {
	x, y    int
	message string
	fg      color.Color
}

type PositionMessage struct {
	worldX, worldY, worldZ int
	message                string
	fg                     color.Color
	ttl                    float64
	ui                     *gfx.Panel
	init                   bool
}

type Runner struct {
	app                  *gfx.App
	ctx                  *bscript.Context
	eventsCall           *bscript.Variable
	deltaArg             *bscript.Value
	fadeDirArg           *bscript.Value
	mouseXArg, mouseYArg *bscript.Value
	sectionLoadCall      *bscript.Variable
	hourArg              *bscript.Value
	hourCall             *bscript.Variable
	sectionLoadXArg      *bscript.Value
	sectionLoadYArg      *bscript.Value
	sectionLoadDataArg   *bscript.Value
	sectionSaveCall      *bscript.Variable
	sectionSaveXArg      *bscript.Value
	sectionSaveYArg      *bscript.Value
	messages             map[int]*Message
	messageIndex         int
	updateOverlay        bool
	Calendar             *Calendar
	positionMessages     []*PositionMessage
	daylight             [24][3]float32
	lastHour             int
}

func NewRunner() *Runner {
	daylight := [24][3]float32{}
	for i := 0; i < 24; i++ {
		daylight[i] = [3]float32{255, 255, 255}
	}
	return &Runner{
		messages:         map[int]*Message{},
		positionMessages: []*PositionMessage{},
		daylight:         daylight,
	}
}

func (runner *Runner) Init(app *gfx.App, config map[string]interface{}) {
	runner.app = app
	if cal, ok := config["calendar"].(map[string]interface{}); ok {
		runner.Calendar = NewCalendar(
			int(cal["min"].(float64)),
			int(cal["hour"].(float64)),
			int(cal["day"].(float64)),
			int(cal["month"].(float64)),
			int(cal["year"].(float64)),
			cal["incrementSpeed"].(float64),
		)

		if daylight, ok := cal["daylight"].(map[string]interface{}); ok {
			for k, v := range daylight {
				hour, err := strconv.Atoi(k)
				if err != nil {
					fmt.Printf("Error parsing daylight hour: %v\n", err)
				} else {
					if hour >= 0 && hour < 24 {
						rgb := v.([]interface{})
						r := util.Clamp(float32(rgb[0].(float64)), 0, 255)
						g := util.Clamp(float32(rgb[1].(float64)), 0, 255)
						b := util.Clamp(float32(rgb[2].(float64)), 0, 255)
						runner.daylight[hour] = [3]float32{r, g, b}
					}
				}
			}
		}
	} else {
		runner.Calendar = NewCalendar(0, 9, 1, 5, 1992, 0.1)
	}
	runner.Calendar.EventListener = runner

	runner.app.Loader.SetIoMode(world.RUNNER_MODE)

	runner.app.Ui.AddBg(0, 0, int(runner.app.Width), int(runner.app.Height), color.Transparent, runner.overlayContents)

	// compile the editor script code
	ast, ctx, err := bscript.Build(
		filepath.Join(app.Config.GameDir, "src", "runner"),
		false,
		map[string]interface{}{
			"app":    app,
			"runner": runner,
		},
	)
	if err != nil {
		panic(err)
	}

	runner.ctx = ctx

	runner.deltaArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.fadeDirArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.mouseXArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.mouseYArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.eventsCall = util.NewFunctionCall("events", runner.deltaArg, runner.fadeDirArg, runner.mouseXArg, runner.mouseYArg)

	runner.hourArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.hourCall = util.NewFunctionCall("onHour", runner.hourArg)

	runner.sectionLoadXArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.sectionLoadYArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.sectionLoadDataArg = &bscript.Value{}
	runner.sectionLoadCall = util.NewFunctionCall("onSectionLoad", runner.sectionLoadXArg, runner.sectionLoadYArg, runner.sectionLoadDataArg)

	runner.sectionSaveXArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.sectionSaveYArg = &bscript.Value{Number: &bscript.SignedNumber{}}
	runner.sectionSaveCall = util.NewFunctionCall("beforeSectionSave", runner.sectionSaveXArg, runner.sectionSaveYArg)

	// run the main method
	_, err = ast.Evaluate(ctx)
	if err != nil {
		panic(err)
	}
}

func (runner *Runner) Name() string {
	return "runner"
}

func (runner *Runner) Events(delta float64, fadeDir int, mouseX, mouseY int32) {
	runner.Calendar.Incr(delta)
	runner.timeoutMessages(delta)
	runner.deltaArg.Number.Number = delta
	runner.fadeDirArg.Number.Number = float64(fadeDir)
	runner.mouseXArg.Number.Number = float64(mouseX)
	runner.mouseYArg.Number.Number = float64(mouseY)
	runner.eventsCall.Evaluate(runner.ctx)
}

func (runner *Runner) GetZ() int {
	return 0
}

func (runner *Runner) SectionLoad(x, y int, data map[string]interface{}) {
	runner.sectionLoadXArg.Number.Number = float64(x)
	runner.sectionLoadYArg.Number.Number = float64(y)
	runner.sectionLoadDataArg.Map = util.ToBscriptMap(data)
	runner.sectionLoadCall.Evaluate(runner.ctx)
}

func (runner *Runner) SectionSave(x, y int) map[string]interface{} {
	runner.sectionSaveXArg.Number.Number = float64(x)
	runner.sectionSaveYArg.Number.Number = float64(y)
	ret, _ := runner.sectionSaveCall.Evaluate(runner.ctx)
	return ret.(map[string]interface{})
}

func (runner *Runner) overlayContents(panel *gfx.Panel) bool {
	if runner.updateOverlay {
		panel.Clear()
		for _, msg := range runner.messages {
			runner.printOutlineMessage(panel, msg.x, msg.y, msg.message, msg.fg)
		}
		runner.updateOverlay = false
		return true
	}
	return false
}

func (runner *Runner) printOutlineMessage(panel *gfx.Panel, x, y int, message string, fg color.Color) {
	for xx := -1; xx <= 1; xx++ {
		for yy := -1; yy <= 1; yy++ {
			runner.app.Font.Printf(panel.Rgba, color.Black, x+xx, y+yy, message)
		}
	}
	runner.app.Font.Printf(panel.Rgba, fg, x, y, message)
}

func (runner *Runner) AddMessage(x, y int, message string, r, g, b uint8) int {
	runner.messages[runner.messageIndex] = &Message{x, y, message, color.RGBA{r, g, b, 255}}
	runner.messageIndex++
	runner.updateOverlay = true
	return runner.messageIndex - 1
}

func (runner *Runner) DelMessage(messageIndex int) {
	delete(runner.messages, messageIndex)
	runner.updateOverlay = true
}

func (runner *Runner) DelAllMessages() {
	runner.messages = map[int]*Message{}
	runner.updateOverlay = true
}

// todo: PositionMessage-s should be vbo-s instead of using the cpu to recalc their positions
const MESSAGE_TTL = 2

func (runner *Runner) ShowMessageAt(worldX, worldY, worldZ int, message string, r, g, b uint8) {
	m := &PositionMessage{
		worldX:  worldX,
		worldY:  worldY,
		worldZ:  worldZ,
		message: message,
		fg:      color.RGBA{r, g, b, 255},
		ttl:     MESSAGE_TTL,
	}
	runner.positionMessages = append(runner.positionMessages, m)
	x, y := runner.app.GetScreenPos(worldX, worldY, worldZ)
	w := runner.app.Font.Width(message)
	m.ui = runner.app.Ui.AddBg(x, y, int(w), runner.app.Font.Height, color.Transparent, func(panel *gfx.Panel) bool {
		if m.init == false {
			panel.Clear()
			runner.printOutlineMessage(panel, 0, int(float32(runner.app.Font.Height)*0.75), m.message, m.fg)
			m.init = true
			return true
		}
		return false
	})
}

func (runner *Runner) timeoutMessages(delta float64) {
	for i, m := range runner.positionMessages {
		m.ttl -= delta
		if m.ttl <= 0 {
			runner.app.Ui.Remove(m.ui)
			runner.positionMessages = append(runner.positionMessages[:i], runner.positionMessages[i+1:]...)
			return
		}
		x, y := runner.app.GetScreenPos(m.worldX, m.worldY, m.worldZ)
		runner.app.Ui.MovePanel(m.ui, x, y)
	}
}

func (runner *Runner) MinsChange(mins, hours, day, month, year int) {
	nowColor := runner.daylight[hours]
	nextHour := hours + 1
	if nextHour >= 24 {
		nextHour -= 24
	}
	nextColor := runner.daylight[nextHour]
	percent := float32(mins) / 60.0
	runner.app.View.SetDaylight(
		util.Linear(nowColor[0], nextColor[0], percent),
		util.Linear(nowColor[1], nextColor[1], percent),
		util.Linear(nowColor[2], nextColor[2], percent),
		255,
	)
	time := ToEpoch(mins, hours, day, month, year)
	if time-runner.lastHour > 60 {
		runner.lastHour = time
		runner.hourArg.Number.Number = float64(hours)
		runner.hourCall.Evaluate(runner.ctx)
	}
}
