package world

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/uzudil/isongn/shapes"
)

const (
	SECTION_SIZE   = 200
	SECTION_Z_SIZE = 24
	VERSION        = 5
	EDITOR_MODE    = 0
	RUNNER_MODE    = 1
)

type SectionPosition struct {
	Block int
	Edge  int
	// extra non blocking shapes: plants, items, etc.
	Extras []int
	Under  int
}

type Section struct {
	X, Y int
	Pos  [SECTION_SIZE][SECTION_SIZE][SECTION_Z_SIZE]SectionPosition
	data map[string]interface{}
}

type SectionCache struct {
	cache [4]*Section
	times [4]float64
}

func NewSectionCache() *SectionCache {
	return &SectionCache{}
}

func (c *SectionCache) describe() string {
	s := ""
	for _, cc := range c.cache {
		if cc == nil {
			s += " __,__"
		} else {
			s += fmt.Sprintf(" %d,%d", cc.X, cc.Y)
		}
	}
	return s
}

type Loader struct {
	observer     WorldObserver
	userDir      string
	gameDir      string
	X, Y         int
	sectionCache *SectionCache
	ioMode       int
}

type WorldObserver interface {
	SectionLoad(x, y int, data map[string]interface{})
	SectionSave(x, y int) map[string]interface{}
	Loading(working bool)
}

func NewLoader(observer WorldObserver, userDir, gameDir string) *Loader {
	return &Loader{observer, userDir, gameDir, 5000, 5000, NewSectionCache(), EDITOR_MODE}
}

func (loader *Loader) SetIoMode(mode int) {
	loader.ioMode = mode
}

func (loader *Loader) IsEditorMode() bool {
	return loader.ioMode == EDITOR_MODE
}

func (loader *Loader) MoveTo(x, y int) bool {
	if x >= 0 && y >= 0 && (loader.X != x || loader.Y != y) {
		loader.X = x
		loader.Y = y
		return true
	}
	return false
}

func (loader *Loader) ClearEdge(x, y int) {
	section, atomX, atomY, _ := loader.getPosInSection(x, y, 0)
	section.Pos[atomX][atomY][0].Edge = 0
}

func (loader *Loader) SetEdge(x, y int, shapeIndex int) {
	section, atomX, atomY, _ := loader.getPosInSection(x, y, 0)
	section.Pos[atomX][atomY][0].Edge = shapeIndex + 1
}

func (loader *Loader) GetEdge(x, y int) (int, bool) {
	section, atomX, atomY, _ := loader.getPosInSection(x, y, 0)
	shapeIndex := section.Pos[atomX][atomY][0].Edge
	if shapeIndex == 0 {
		return 0, false
	}
	return shapeIndex - 1, true
}

func (loader *Loader) SetShape(x, y, z int, shapeIndex int) bool {
	section, atomX, atomY, atomZ := loader.getPosInSection(x, y, z)
	section.Pos[atomX][atomY][atomZ].Block = shapeIndex + 1
	return true
}

func (loader *Loader) EraseShape(x, y, z int) bool {
	section, atomX, atomY, atomZ := loader.getPosInSection(x, y, z)
	shapeIndex := section.Pos[atomX][atomY][atomZ].Block
	if shapeIndex > 0 {
		section.Pos[atomX][atomY][atomZ].Block = 0
		return true
	}
	return false
}

func (loader *Loader) GetPos(worldX, worldY, worldZ int) *SectionPosition {
	section, atomX, atomY, atomZ := loader.getPosInSection(worldX, worldY, worldZ)
	return &section.Pos[atomX][atomY][atomZ]
}

func (loader *Loader) GetShape(worldX, worldY, worldZ int) (int, bool) {
	section, atomX, atomY, atomZ := loader.getPosInSection(worldX, worldY, worldZ)
	shapeIndex := section.Pos[atomX][atomY][atomZ].Block
	if shapeIndex == 0 {
		return 0, false
	}
	return shapeIndex - 1, true
}

func (loader *Loader) AddExtra(x, y, z int, shapeIndex int) bool {
	section, atomX, atomY, atomZ := loader.getPosInSection(x, y, z)
	section.Pos[atomX][atomY][atomZ].Extras = append(section.Pos[atomX][atomY][atomZ].Extras, shapeIndex)
	return true
}

func (loader *Loader) EraseExtra(x, y, z, shapeIndex int) bool {
	section, atomX, atomY, atomZ := loader.getPosInSection(x, y, z)
	e := section.Pos[atomX][atomY][atomZ].Extras
	for index, currShapeIndex := range e {
		if currShapeIndex == shapeIndex {
			section.Pos[atomX][atomY][atomZ].Extras = append(e[:index], e[index+1:]...)
			return true
		}
	}
	return false
}

func (loader *Loader) EraseAllExtras(x, y, z int) bool {
	section, atomX, atomY, atomZ := loader.getPosInSection(x, y, z)
	section.Pos[atomX][atomY][atomZ].Extras = []int{}
	return true
}

func (loader *Loader) GetExtras(worldX, worldY, worldZ int) []int {
	section, atomX, atomY, atomZ := loader.getPosInSection(worldX, worldY, worldZ)
	return section.Pos[atomX][atomY][atomZ].Extras
}

func (loader *Loader) GetExtra(worldX, worldY, worldZ, i int) (int, bool) {
	section, atomX, atomY, atomZ := loader.getPosInSection(worldX, worldY, worldZ)
	if i < len(section.Pos[atomX][atomY][atomZ].Extras) {
		return section.Pos[atomX][atomY][atomZ].Extras[i], true
	}
	return 0, false
}

func (loader *Loader) GetSectionPos() (int, int) {
	sx := loader.X / SECTION_SIZE
	sy := loader.Y / SECTION_SIZE
	return sx, sy
}

func (loader *Loader) getPosInSection(worldX, worldY, worldZ int) (*Section, int, int, int) {
	sx := worldX / SECTION_SIZE
	sy := worldY / SECTION_SIZE
	section, err := loader.getSection(sx, sy)
	if err != nil {
		log.Fatal(err)
	}
	atomX := worldX % SECTION_SIZE
	atomY := worldY % SECTION_SIZE
	atomZ := worldZ
	return section, atomX, atomY, atomZ
}

func (loader *Loader) getSection(sx, sy int) (*Section, error) {
	// already loaded?
	nilFound := false
	oldestIndex := -1
	for i := 0; i < len(loader.sectionCache.cache); i++ {
		if loader.sectionCache.cache[i] == nil {
			nilFound = true
			oldestIndex = i
		} else {
			if loader.sectionCache.cache[i].X == sx && loader.sectionCache.cache[i].Y == sy {
				return loader.sectionCache.cache[i], nil
			}
		}
	}

	px := loader.X / SECTION_SIZE
	py := loader.Y / SECTION_SIZE
	if nilFound == false {
		oldestIndex = -1
		for i := 0; i < len(loader.sectionCache.cache); i++ {
			c := loader.sectionCache.cache[i]
			if (px != c.X || py != c.Y) && (oldestIndex == -1 || loader.sectionCache.times[i] < loader.sectionCache.times[oldestIndex]) {
				oldestIndex = i
			}
		}
	}

	loader.observer.Loading(true)

	// save version in cache
	if loader.sectionCache.cache[oldestIndex] != nil {
		oldSection := loader.sectionCache.cache[oldestIndex]
		fmt.Printf("+++ NEED section %d,%d, EVICTING %d,%d, PLAYER in %d,%d CACHE=%s\n",
			sx, sy,
			oldSection.X, oldSection.Y,
			px, py,
			loader.sectionCache.describe(),
		)
		oldSection.data = loader.observer.SectionSave(oldSection.X, oldSection.Y)
		err := loader.save(oldSection)
		if err != nil {
			return nil, err
		}
	}

	// not found in cache, load it
	section, err := loader.load(sx, sy)
	if err != nil {
		return nil, err
	}

	// put in cache
	loader.sectionCache.cache[oldestIndex] = section
	loader.sectionCache.times[oldestIndex] = glfw.GetTime()

	loader.observer.SectionLoad(sx, sy, section.data)
	loader.observer.Loading(false)

	// fmt.Printf("CACHE: \n")
	// for i := range loader.sectionCache.cache {
	// 	section := loader.sectionCache.cache[i]
	// 	if section == nil {
	// 		fmt.Printf("\tnil\n")
	// 	} else {
	// 		fmt.Printf("\t%d,%d %f\n", section.X, section.Y, loader.sectionCache.times[i])
	// 	}
	// }
	// fmt.Printf("---------------\n")

	return section, nil
}

func (loader *Loader) SaveAll() error {
	for _, c := range loader.sectionCache.cache {
		if c != nil {
			c.data = loader.observer.SectionSave(c.X, c.Y)
			err := loader.save(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (loader *Loader) load(sx, sy int) (*Section, error) {
	section := &Section{
		X:    sx,
		Y:    sy,
		data: map[string]interface{}{},
	}

	mapName := mapFileName(sx, sy)
	var path string
	if loader.ioMode == EDITOR_MODE {
		// the editor io is always from the game dir
		path = filepath.Join(loader.gameDir, "maps", mapName)
	} else {
		// the runner io tries from user dir
		path = filepath.Join(loader.userDir, mapName)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// and if that fails, from game dir
			path = filepath.Join(loader.gameDir, "maps", mapName)
		}
	}

	if _, err := os.Stat(path); err == nil {
		defer un(trace(fmt.Sprintf("Loading map %d,%d", sx, sy)))

		f, err := os.Open(path)
		defer f.Close()
		if err != nil {
			return nil, err
		}

		fz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer fz.Close()

		version := make([]byte, 1)
		_, err = fz.Read(version)
		if err != nil {
			return nil, err
		}

		dec := gob.NewDecoder(fz)
		err = dec.Decode(&section.Pos)
		if err != nil {
			return nil, err
		}
		if version[0] >= 3 {
			var bytes []byte
			err = dec.Decode(&bytes)
			if err != nil {
				return nil, err
			}
			data := map[string]interface{}{}
			err = json.Unmarshal(bytes, &data)
			if err != nil {
				return nil, err
			}
			fixArrays(data)
			section.data = data
		}
	}
	return section, nil
}

func (section *Section) calculateUnder() {
	// configure Under[]
	for x := 0; x < SECTION_SIZE; x++ {
		for y := 0; y < SECTION_SIZE; y++ {
			for z := 0; z < SECTION_Z_SIZE-1; z++ {
				section.Pos[x][y][z].Under = 0
				for zz := z + 1; zz < SECTION_Z_SIZE; zz++ {
					// todo: 4x4 grid used below should be in config.json
					cover := section.Pos[(x/4)*4][(y/4)*4][zz].Block
					if cover > 0 && shapes.Shapes[cover-1].Group > 0 {
						section.Pos[x][y][z].Under = cover
						break
					}
				}
			}
		}
	}
}

func (loader *Loader) save(section *Section) error {
	defer un(trace(fmt.Sprintf("Saving map %d,%d", section.X, section.Y)))

	mapName := mapFileName(section.X, section.Y)
	var path string
	if loader.ioMode == EDITOR_MODE {
		// the editor io is always to the game dir
		path = filepath.Join(loader.gameDir, "maps", mapName)
	} else {
		// the runner io always to user dir
		path = filepath.Join(loader.userDir, mapName)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fz := gzip.NewWriter(f)
	defer fz.Close()

	b := []byte{VERSION}
	fz.Write(b)

	if loader.ioMode == EDITOR_MODE {
		section.calculateUnder()
	}

	enc := gob.NewEncoder(fz)
	err = enc.Encode(section.Pos)
	if err != nil {
		return err
	}
	jsonstr, err := json.Marshal(section.data)
	if err != nil {
		return err
	}
	err = enc.Encode([]byte(jsonstr))
	if err != nil {
		return err
	}

	return nil
}

func fixArrays(data interface{}) {

	mapdata, ok := data.(map[string]interface{})
	if ok {
		for k, v := range mapdata {
			m, ok := v.(map[string]interface{})
			if ok {
				fixArrays(m)
			}

			arr, ok := v.([]interface{})
			if ok {
				fixArrays(arr)
				mapdata[k] = &arr
			}
		}
	}

	arrdata, ok := data.([]interface{})
	if ok {
		for i, v := range arrdata {
			m, ok := v.(map[string]interface{})
			if ok {
				fixArrays(m)
			}

			arr, ok := v.([]interface{})
			if ok {
				fixArrays(arr)
				arrdata[i] = &arr
			}
		}
	}
}

func trace(s string) (string, time.Time) {
	return s, time.Now()
}

func un(s string, startTime time.Time) {
	endTime := time.Now()
	log.Println(s, "ElapsedTime in seconds:", endTime.Sub(startTime))
}

func mapFileName(sx, sy int) string {
	return fmt.Sprintf("map%02x%02x", sx, sy)
}
