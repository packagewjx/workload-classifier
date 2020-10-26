package alitrace

import (
	"bufio"
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
)

func SplitContainerUsage(in io.Reader, meta map[string][]*ContainerMeta) error {
	const UnknownApp = "unknown"
	type Context struct {
		file   *os.File
		writer *bufio.Writer
	}

	reader := bufio.NewReader(in)

	writerMap := make(map[string]*Context)

	var line string
	var err error
	for line, err = reader.ReadString(core.LineBreak); err == nil || (err == io.EOF && line != ""); line, err = reader.ReadString(core.LineBreak) {
		if line == "" {
			continue
		}
		cid := line[:strings.Index(line, core.Splitter)]

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
	}

	if err != io.EOF {
		return errors.Wrap(err, "读取监控数据出错")
	}

	// 写入剩余的
	for _, ctx := range writerMap {
		_ = ctx.writer.Flush()
		_ = ctx.file.Close()
	}

	return nil
}
