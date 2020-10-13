package alitrace

import (
	"bufio"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"io"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const StatusStarted = "started"

type ContainerWorkloadReader interface {
	Read() ([]*internal.ContainerWorkloadData, error)
}

func NewContainerWorkloadReader(in io.Reader, meta map[string][]*ContainerMeta) ContainerWorkloadReader {
	return &containerWorkloadReader{
		reader: bufio.NewReader(in),
		meta:   meta,
	}
}

type containerWorkloadReader struct {
	reader    *bufio.Reader
	meta      map[string][]*ContainerMeta
	ReadCount uint64
}

type rawSectionData struct {
	cpu    []float32
	mem    []float32
	cpuSum float32
	memSum float32
}

func (u *containerWorkloadReader) Read() ([]*internal.ContainerWorkloadData, error) {
	rawData, err := u.readRawData()
	if err != nil {
		return nil, err
	}

	return u.processRawData(rawData), nil
}

func (u *containerWorkloadReader) readRawData() (map[string][]*rawSectionData, error) {
	containerSection := make(map[string][]*rawSectionData)

	// 读取输入
	var line string
	var err error
	lineNum := 0
	for line, err = u.reader.ReadString(internal.LineBreak); err == nil || (line != "" && err == io.EOF); line, err = u.reader.ReadString(internal.LineBreak) {
		lineNum++
		u.ReadCount += uint64(len(line))
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		record := strings.Split(line, internal.Splitter)

		cid := record[0]
		timestamp, _ := strconv.ParseInt(record[2], 10, 64)
		statuses := u.meta[cid]
		// statusIdx应该是第一个大于等于timestamp的记录。这里要的应该是前一条记录，因为前一条记录的元数据信息才是本时间点的元数据
		statusIdx := sort.Search(len(statuses), func(i int) bool {
			return statuses[i].Timestamp >= timestamp
		})
		if len(statuses) == 0 {
			log.Printf("没有containerID为'%s'的元数据记录，行号%d\n", cid, lineNum)
			continue
		} else if statusIdx == 0 {
			log.Printf("不应该找到位置为0的记录，containerID: %s，行号：%d\n", cid, lineNum)
		} else {
			statusIdx--
		}

		if statuses[statusIdx].Status != StatusStarted {
			log.Printf("找到的状态不是开始，containerId：%s，行号%d\n", cid, lineNum)
		}

		// statuses留作后用

		sectionIndex := timestamp % internal.DayLength / internal.SectionLength
		cpuUtil, _ := strconv.ParseFloat(record[3], 32)
		memUtil, _ := strconv.ParseFloat(record[4], 32)
		sections, ok := containerSection[cid]
		if !ok {
			sections = make([]*rawSectionData, internal.NumSections)
			for i := 0; i < len(sections); i++ {
				sections[i] = &rawSectionData{
					cpu: make([]float32, 0, 128),
					mem: make([]float32, 0, 128),
				}
			}
			containerSection[cid] = sections
		}

		cpu := cpuUtil
		mem := memUtil

		sect := sections[sectionIndex]
		sect.cpu = append(sect.cpu, float32(cpu))
		sect.mem = append(sect.mem, float32(mem))
		sect.cpuSum += float32(cpu)
		sect.memSum += float32(mem)
	}
	return containerSection, nil
}

func (u *containerWorkloadReader) processRawData(containerSection map[string][]*rawSectionData) []*internal.ContainerWorkloadData {
	// 处理数据
	cDataArray := make([]*internal.ContainerWorkloadData, len(containerSection))
	idx := 0
	wg := sync.WaitGroup{}
	for cid, sections := range containerSection {
		wg.Add(1)
		go func(i int, containerId string, rawData []*rawSectionData) {
			defer wg.Done()
			cData := &internal.ContainerWorkloadData{
				ContainerId: cid,
				Data:        make([]*internal.SectionData, len(rawData)),
			}

			for si, section := range rawData {
				stat := &internal.SectionData{}
				if len(section.cpu) != 0 {
					stat.CpuAvg = section.cpuSum / float32(len(section.cpu))
					stat.CpuMax = utils.GetSortedPositionValue(section.cpu, len(section.cpu)-1)
					stat.CpuMin = utils.GetSortedPositionValue(section.cpu, 0)
					stat.CpuP50 = utils.GetSortedPositionValue(section.cpu, len(section.cpu)/2)
					stat.CpuP90 = utils.GetSortedPositionValue(section.cpu, len(section.cpu)*90/100)
					stat.CpuP99 = utils.GetSortedPositionValue(section.cpu, len(section.cpu)*99/100)
				} else {
					stat.CpuAvg = float32(math.NaN())
					stat.CpuMax = float32(math.NaN())
					stat.CpuMin = float32(math.NaN())
					stat.CpuP50 = float32(math.NaN())
					stat.CpuP90 = float32(math.NaN())
					stat.CpuP99 = float32(math.NaN())
				}
				if len(section.mem) != 0 {
					stat.MemAvg = section.memSum / float32(len(section.mem))
					stat.MemMax = utils.GetSortedPositionValue(section.mem, len(section.mem)-1)
					stat.MemMin = utils.GetSortedPositionValue(section.mem, 0)
					stat.MemP50 = utils.GetSortedPositionValue(section.mem, len(section.mem)/2)
					stat.MemP90 = utils.GetSortedPositionValue(section.mem, len(section.mem)*90/100)
					stat.MemP99 = utils.GetSortedPositionValue(section.mem, len(section.mem)*99/100)
				} else {
					stat.MemAvg = float32(math.NaN())
					stat.MemMax = float32(math.NaN())
					stat.MemMin = float32(math.NaN())
					stat.MemP50 = float32(math.NaN())
					stat.MemP90 = float32(math.NaN())
					stat.MemP99 = float32(math.NaN())
				}
				cData.Data[si] = stat
			}
			cDataArray[i] = cData
		}(idx, cid, sections)
		idx++
	}
	wg.Wait()
	return cDataArray
}
