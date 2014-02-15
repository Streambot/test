package streambot

import(
	"sync"
	"math"
	"math/rand"
	"time"
	"fmt"
)

func RandInt(min int , max int) int {
    return min + rand.Intn(max-min)
}

type Sampler struct {
	SampleRate			float64
	MinuteSlices		map[int64][]string
	MinuteSlicesMutex	*sync.RWMutex
	Minutes 			[]int64
	Stats				*Statter
}

func NewSampler(sampleRate float64, stats *Statter) *Sampler {
	s := new(Sampler)
	s.MinuteSlices = map[int64][]string{}
	s.MinuteSlicesMutex = new(sync.RWMutex)
	s.Minutes = []int64{}
	s.SampleRate = sampleRate
	s.Stats = stats
	return s
}

func (sampler *Sampler)ScrapMinuteSlice(minuteSliceKey int64) {
	slice := sampler.MinuteSlices[minuteSliceKey]
	if slice == nil {
		return
	}
	sampler.Stats.Count("sampling.scrap_minute_slice")
	sliceLen := len(slice)
	numSelects := int(math.Floor((float64(sliceLen) * sampler.SampleRate) + .5))
	offset := int(math.Floor((float64(sliceLen)/float64(numSelects)) + .5))
	idx := offset
	var tmpSlice []string
	for idx < sliceLen {
		tmpSlice = append(tmpSlice, slice[idx])
		idx = idx + offset
	}
	sampler.MinuteSlices[minuteSliceKey] = tmpSlice
    return 
}

func (sampler *Sampler)SampleId(id string) {
	sampler.Stats.Count("sampling.sample")
	sampler.MinuteSlicesMutex.Lock()
	minuteSliceKey := time.Now().Unix()/60
	if len(sampler.MinuteSlices[minuteSliceKey]) == 0 {
		sampler.ScrapMinuteSlice(minuteSliceKey - 1)
		sampler.Minutes = append(sampler.Minutes, minuteSliceKey)
	}
	sampler.MinuteSlices[minuteSliceKey] = append(sampler.MinuteSlices[minuteSliceKey], id)
	sampler.MinuteSlicesMutex.Unlock()
}

func (sampler *Sampler)RandomSampledId() (id string) {
	fmt.Println(fmt.Sprintf("%d minute slices", len(sampler.Minutes)))
	if len(sampler.Minutes) == 0 {
		return
	}
	minute := sampler.Minutes[int64(RandInt(0, len(sampler.Minutes)))]
	slice := sampler.MinuteSlices[minute]
	fmt.Println(fmt.Sprintf("%d entries", len(slice)))
	if len(slice) == 0 {
		return
	}
	id = slice[int64(RandInt(0, len(slice)))]
	return
}