package alitrace

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const SectionLength = 15 * 60
const DayLength = 24 * 60 * 60
const NumSections = DayLength / SectionLength
const NumSectionFields = 12
const LineBreak = '\n'
const Splitter = ","
const PreProcessOutputSuffix = ".preprocessed.csv"
const StatusStarted = "started"

type ContainerStatus struct {
	ContainerId string
	MachineId   string
	Timestamp   int64
	AppDu       string
	Status      string
	CpuRequest  int
	CpuLimit    int
	MemSize     float32
}

func LoadContainerMeta(file string) (map[string][]*ContainerStatus, error) {
	fin, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "打开文件错误")
	}
	defer func() {
		_ = fin.Close()
	}()

	reader := csv.NewReader(fin)

	containerMeta := make(map[string][]*ContainerStatus)
	var record []string
	cnt := 0
	for record, err = reader.Read(); err == nil; record, err = reader.Read() {
		cnt++
		cid := record[0]
		s, ok := containerMeta[cid]
		if !ok {
			s = make([]*ContainerStatus, 0, 10)
		}
		timestamp, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的timestamp格式错误，记录为%s，错误为%v\n", cnt, record[2], err)
		}
		cpuRequest, err := strconv.ParseInt(record[5], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的cpu request格式错误，记录为%s，错误为%v\n", cnt, record[5], err)
		}
		cpuLimit, err := strconv.ParseInt(record[6], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的cpu limit格式错误，记录为%s，错误为%v\n", cnt, record[6], err)
		}
		memSize, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			log.Printf("第%d条记录的mem size格式错误，记录为%s，错误为%v\n", cnt, record[7], err)
		}

		s = append(s, &ContainerStatus{
			ContainerId: cid,
			MachineId:   record[1],
			Timestamp:   timestamp,
			AppDu:       record[3],
			Status:      record[4],
			CpuRequest:  int(cpuRequest),
			CpuLimit:    int(cpuLimit),
			MemSize:     float32(memSize),
		})
		containerMeta[cid] = s
	}
	if err != io.EOF {
		return nil, errors.Wrap(err, "读取元数据错误")
	}

	return containerMeta, nil
}

func SplitContainerUsage(fileName string, meta map[string][]*ContainerStatus, lineCount *int) error {
	const UnknownApp = "unknown"
	type Context struct {
		file   *os.File
		writer *bufio.Writer
	}

	file, err := os.Open(fileName)
	if err != nil {
		return errors.Wrap(err, "打开容器监控失败")
	}
	fin := bufio.NewReader(file)

	writerMap := make(map[string]*Context)

	var line string
	for line, err = fin.ReadString(LineBreak); err == nil; line, err = fin.ReadString(LineBreak) {
		if line == "" {
			continue
		}
		cid := line[:strings.Index(line, Splitter)]

		var appDu string
		record, ok := meta[cid]
		if !ok || len(record) == 0 {
			appDu = UnknownApp
		} else {
			appDu = record[0].AppDu
		}
		ctx, ok := writerMap[appDu]
		if !ok {
			fileName := appDu + ".csv"
			file, err := os.Create(fileName)
			if err != nil {
				return errors.Wrap(err, "创建"+fileName+"失败")
			}
			writer := bufio.NewWriter(file)
			ctx = &Context{
				file:   file,
				writer: writer,
			}
			writerMap[appDu] = ctx
		}

		n, err := ctx.writer.WriteString(line)
		if n != len(line) {
			return errors.New(fmt.Sprintf("实际写入字节过少。实际数：%d，应该为：%d，文件名：%s.csv", n, len(line), appDu))
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("写入%s.csv发生错误", appDu))
		}

		*lineCount++
	}

	if err != io.EOF {
		return errors.Wrap(err, "读取监控数据出错")
	}

	// 写入剩余的
	for _, ctx := range writerMap {
		_ = ctx.writer.Flush()
		_ = ctx.file.Close()
	}

	_ = file.Close()
	return nil
}

type sectionData struct {
	cpu    []float32
	mem    []float32
	cpuSum float32
	memSum float32
}

func parseUsageData(inFile string, byteRead *int64, meta map[string][]*ContainerStatus, logger *log.Logger) (map[string][]*sectionData, error) {
	containerSection := make(map[string][]*sectionData)
	fin, err := os.Open(inFile)
	if err != nil {
		return nil, errors.Wrap(err, "打开输入文件失败")
	}
	reader := bufio.NewReader(fin)

	defer func() {
		_ = fin.Close()
	}()

	// 读取输入
	var line string
	lineNum := 0
	for line, err = reader.ReadString(LineBreak); err == nil; line, err = reader.ReadString(LineBreak) {
		lineNum++
		*byteRead += int64(len(line))
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		record := strings.Split(line, Splitter)

		cid := record[0]
		timestamp, _ := strconv.ParseInt(record[2], 10, 64)
		statuses := meta[cid]
		// statusIdx应该是第一个大于等于timestamp的记录。这里要的应该是前一条记录，因为前一条记录的元数据信息才是本时间点的元数据
		statusIdx := sort.Search(len(statuses), func(i int) bool {
			return statuses[i].Timestamp >= timestamp
		})
		if len(statuses) == 0 {
			logger.Printf("没有containerID为'%s'的元数据记录，行号%d\n", cid, lineNum)
			continue
		} else if statusIdx == 0 {
			logger.Printf("不应该找到位置为0的记录，containerID: %s，行号：%d\n", cid, lineNum)
		} else {
			statusIdx--
		}

		if statuses[statusIdx].Status != StatusStarted {
			logger.Printf("找到的状态不是开始，containerId：%s，行号%d\n", cid, lineNum)
		}

		sectionIndex := timestamp % DayLength / SectionLength
		cpuUtil, _ := strconv.ParseFloat(record[3], 32)
		memUtil, _ := strconv.ParseFloat(record[4], 32)
		sections, ok := containerSection[cid]
		if !ok {
			sections = make([]*sectionData, NumSections)
			for i := 0; i < len(sections); i++ {
				sections[i] = &sectionData{
					cpu: make([]float32, 0, 128),
					mem: make([]float32, 0, 128),
				}
			}
			containerSection[cid] = sections
		}

		cpu := cpuUtil / 100.0 * float64(statuses[statusIdx].CpuRequest)
		mem := memUtil / 100.0 * float64(statuses[statusIdx].MemSize)

		sect := sections[sectionIndex]
		sect.cpu = append(sect.cpu, float32(cpu))
		sect.mem = append(sect.mem, float32(mem))
		sect.cpuSum += float32(cpu)
		sect.memSum += float32(mem)
	}
	_ = fin.Close()
	fin = nil
	return containerSection, nil
}

func outputProcessedData(outFile string, outputHeader bool, processedSections *sync.Map) error {
	// 输出数据
	fout, err := os.Create(outFile)
	if err != nil {
		return errors.Wrap(err, "创建输出文件失败")
	}
	writer := csv.NewWriter(fout)

	defer func() {
		writer.Flush()
		_ = fout.Close()
	}()

	// 输出表头
	if outputHeader {
		header := make([]string, 1, 1153)
		header[0] = "container_id"
		for i := 0; i < NumSections; i++ {
			header = append(header,
				fmt.Sprintf("cpu_avg_%d", i),
				fmt.Sprintf("cpu_max_%d", i),
				fmt.Sprintf("cpu_min_%d", i),
				fmt.Sprintf("cpu_p50_%d", i),
				fmt.Sprintf("cpu_p90_%d", i),
				fmt.Sprintf("cpu_p99_%d", i),
				fmt.Sprintf("mem_avg_%d", i),
				fmt.Sprintf("mem_max_%d", i),
				fmt.Sprintf("mem_min_%d", i),
				fmt.Sprintf("mem_p50_%d", i),
				fmt.Sprintf("mem_p90_%d", i),
				fmt.Sprintf("mem_p99_%d", i))
		}
		err := writer.Write(header)
		if err != nil {
			return errors.Wrap(err, "写入表头错误")
		}
	}

	err = nil
	processedSections.Range(func(key, value interface{}) bool {
		cid := key.(string)
		processedSectionDataArray := value.([]*internal.ProcessedSectionData)

		record := make([]string, 1, 1+96*12)
		record[0] = cid
		for i := 0; i < len(processedSectionDataArray); i++ {
			stat := processedSectionDataArray[i]
			record = append(record, fmt.Sprintf("%.2f", stat.CpuAvg))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuMax))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuMin))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP50))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP90))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP99))
			record = append(record, fmt.Sprintf("%.2f", stat.MemAvg))
			record = append(record, fmt.Sprintf("%.2f", stat.MemMax))
			record = append(record, fmt.Sprintf("%.2f", stat.MemMin))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP50))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP90))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP99))
		}
		err2 := writer.Write(record)
		if err2 != nil {
			err = errors.Wrap(err2, "写入"+cid+"数据错误")
			return false
		}
		return true
	})

	if err != nil {
		return err
	}

	return nil
}

func processSectionData(containerSection map[string][]*sectionData) *sync.Map {
	// 处理数据
	processedSections := &sync.Map{}
	for cid, sections := range containerSection {
		processedSectionDataArray := make([]*internal.ProcessedSectionData, len(sections))
		for i, section := range sections {
			stat := &internal.ProcessedSectionData{}
			if len(section.cpu) != 0 {
				stat.CpuAvg = section.cpuSum / float32(len(section.cpu))
				stat.CpuMax = getSortedPositionValue(section.cpu, len(section.cpu)-1)
				stat.CpuMin = getSortedPositionValue(section.cpu, 0)
				stat.CpuP50 = getSortedPositionValue(section.cpu, len(section.cpu)/2)
				stat.CpuP90 = getSortedPositionValue(section.cpu, len(section.cpu)*90/100)
				stat.CpuP99 = getSortedPositionValue(section.cpu, len(section.cpu)*99/100)
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
				stat.MemMax = getSortedPositionValue(section.mem, len(section.mem)-1)
				stat.MemMin = getSortedPositionValue(section.mem, 0)
				stat.MemP50 = getSortedPositionValue(section.mem, len(section.mem)/2)
				stat.MemP90 = getSortedPositionValue(section.mem, len(section.mem)*90/100)
				stat.MemP99 = getSortedPositionValue(section.mem, len(section.mem)*99/100)
			} else {
				stat.MemAvg = float32(math.NaN())
				stat.MemMax = float32(math.NaN())
				stat.MemMin = float32(math.NaN())
				stat.MemP50 = float32(math.NaN())
				stat.MemP90 = float32(math.NaN())
				stat.MemP99 = float32(math.NaN())
			}
			processedSectionDataArray[i] = stat
		}
		processedSections.Store(cid, processedSectionDataArray)
	}
	return processedSections
}

// output file format
// container_id, cpuAvg_1, cpuMax_1, cpuMin_1, cpuP50_1, cpuP90_1, cpuP99_1, memAvg_1, memMax_1, memMin_1, memP50_1, memP90_1, memP99_1 ... (from 1 to 96)
func PreProcessUsages(inFiles []string, meta map[string][]*ContainerStatus, byteRead *int64, outputHeader bool, numRoutine int) error {
	logFile, err := os.Create("preprocess.log")
	if err != nil {
		return errors.Wrap(err, "创建日志文件失败")
	}
	logger := log.New(logFile, "", log.LstdFlags|log.Lmsgprefix|log.Lshortfile)

	wg := sync.WaitGroup{}
	sem := make(chan struct{}, numRoutine)
	errCh := make(chan error, numRoutine)
	errArr := make([]error, 0, 10)
	doneCh := make(chan struct{})

	defer func() {
		_ = logFile.Close()
		close(sem)
		close(errCh)
		close(doneCh)
	}()

	go func() {
		select {
		case err := <-errCh:
			errArr = append(errArr, err)
		case <-doneCh:
			return
		}
	}()

	for _, f := range inFiles {
		wg.Add(1)
		go func(fileName string) {
			defer wg.Done()
			defer func() {
				<-sem
			}()
			sem <- struct{}{}

			containerSection, err := parseUsageData(fileName, byteRead, meta, logger)
			if err != nil {
				errCh <- err
				return
			}

			processedSections := processSectionData(containerSection)

			err = outputProcessedData(fileName+PreProcessOutputSuffix, outputHeader, processedSections)
			if err != nil {
				errCh <- err
				return
			}
		}(f)
	}

	wg.Wait()
	doneCh <- struct{}{}

	if len(errArr) > 0 {
		errMsg := ""
		for i, err := range errArr {
			errMsg += fmt.Sprintf("err %d:\n %v", i, err)
		}
		return errors.New(errMsg)
	}

	return nil
}

func PreProcessUsagesAndMerge(inFiles []string, outFile string, meta map[string][]*ContainerStatus, outputHeader bool, byteRead *int64, numRoutine int) error {
	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		return fmt.Errorf("无法合并，输出文件%s已存在\n", outFile)
	}

	err := PreProcessUsages(inFiles, meta, byteRead, false, numRoutine)
	if err != nil {
		return err
	}

	for i := 0; i < len(inFiles); i++ {
		inFiles[i] = inFiles[i] + PreProcessOutputSuffix
	}

	// 表头输出
	if outputHeader {
		const HeaderFile = "header.csv"
		err = outputProcessedData(HeaderFile, true, &sync.Map{})
		if err != nil {
			return errors.Wrap(err, "输出表头错误")
		}
		inFiles = append([]string{HeaderFile}, inFiles...)
	}

	err = mergeFile(inFiles, outFile)
	if err != nil {
		return err
	}

	// 删除临时输出
	for _, file := range inFiles {
		err := os.Remove(file)
		if err != nil {
			return errors.Wrap(err, "删除临时文件失败")
		}
	}

	return nil
}

func mergeFile(inFiles []string, outFile string) error {
	fout, err := os.Create(outFile)
	if err != nil {
		return errors.Wrap(err, "创建合并输出文件失败")
	}
	defer func() {
		_ = fout.Close()
	}()

	buf := make([]byte, 4096)

	for _, fileName := range inFiles {
		fin, err := os.Open(fileName)
		if err != nil {
			return errors.Wrap(err, "打开预处理文件"+fileName+"失败")
		}
		var read int
		for read, err = fin.Read(buf); read > 0; read, err = fin.Read(buf) {
			write, err := fout.Write(buf[:read])
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("读取文件%s过程中，输出到合并文件失败", fileName))
			}
			if write < read {
				return fmt.Errorf("读取文件%s过程中，输出字节不足，应该为%d，实际为%d", fileName, read, write)
			}
		}
		if err != io.EOF {
			return errors.Wrap(err, fmt.Sprintf("读取文件%s失败", fileName))
		}

		_ = fin.Close()
	}

	return nil
}

func imputeData(record []string) {
	recordIndex := func(fieldIndex, sectionIndex int) int {
		return sectionIndex*NumSectionFields + fieldIndex
	}
	imputeFunc := func(start, end float64, fieldIndex, leftInclusive, rightInclusive int) {
		// +2 是因为需要保证左右有效值就是start和end，而不是第一个和最后一个NaN是start和end
		// data[leftInclusive-1]=start data[rightInclusive+1]=end
		k := (end - start) / float64(rightInclusive-leftInclusive+2)
		for i := leftInclusive; i <= rightInclusive; i++ {
			record[recordIndex(fieldIndex, i)] = fmt.Sprintf("%.2f", start+k*float64(i-(leftInclusive-1)))
		}
	}

	for i := 0; i < NumSectionFields; i++ {
		invalidLeft := -1
		for j := 0; j < NumSections; j++ {
			if record[recordIndex(i, j)] == "NaN" {
				if invalidLeft == -1 {
					invalidLeft = j
				}
			} else {
				if invalidLeft != -1 {
					startVal := 0.0
					if invalidLeft != 0 {
						startVal, _ = strconv.ParseFloat(record[recordIndex(i, invalidLeft-1)], 64)
					}
					endVal, _ := strconv.ParseFloat(record[recordIndex(i, j)], 64)

					imputeFunc(startVal, endVal, i, invalidLeft, j-1)
					invalidLeft = -1
				}
			}
		}

		// 检查最后的区间是否为NaN
		if invalidLeft != -1 {
			if invalidLeft == 0 {
				// 这种情况是整段数据都为NaN，暂时没有办法填充
			} else {
				startVal, _ := strconv.ParseFloat(record[recordIndex(i, invalidLeft-1)], 64)
				imputeFunc(startVal, 0, i, invalidLeft, NumSections-1)
			}
		}
	}
}

func ImputeMissingValues(inFile string, outFile string) error {
	fin, err := os.Open(inFile)
	if err != nil {
		return errors.Wrap(err, "打开输入文件出错")
	}
	reader := bufio.NewReader(fin)
	fout, err := os.Create(outFile)
	if err != nil {
		return errors.Wrap(err, "创建输出文件出错")
	}
	writer := bufio.NewWriter(fout)
	defer func() {
		_ = fin.Close()
		_ = writer.Flush()
		_ = fout.Close()
	}()

	var line string
	lineCount := 0
	for line, err = reader.ReadString(LineBreak); err == nil; line, err = reader.ReadString(LineBreak) {
		lineCount++
		if strings.Contains(line, "NaN") {
			log.Printf("第%d行记录有NaN值，正在插值\n", lineCount)

			record := strings.Split(strings.TrimSpace(line), Splitter)
			if len(record) < NumSections*NumSectionFields {
				return errors.New("文件记录格式不对")
			}
			startPos := len(record) - NumSections*NumSectionFields

			imputeData(record[startPos:])

			line = strings.Join(record, Splitter) + string(LineBreak)
		}
		n, err := writer.WriteString(line)
		if err != nil {
			return errors.Wrap(err, "输出文件错误")
		}
		if n != len(line) {
			return errors.New("输出不足")
		}
	}

	return nil
}
