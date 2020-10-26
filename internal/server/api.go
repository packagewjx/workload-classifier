package server

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"reflect"
)

func (s *serverImpl) QueryAppCharacteristics(appName server.AppName) (*server.AppCharacteristics, error) {
	s.logger.Printf("接收到查询名称空间为%s，名称为%s的请求\n", appName.Namespace, appName.Name)
	appClass, err := s.dao.QueryAppClassByApp(&appName)
	if err == server.ErrAppNotFound || err == server.ErrAppNotClassified {
		return nil, err
	} else if err != nil {
		s.logger.Printf("查询AppClass失败，原因为：%v\n", err)
		return nil, err
	}

	metric, err := s.dao.QueryClassMetricsByClassId(appClass.ClassId)
	if err != nil {
		s.logger.Printf("查询ClassMetrics时出错，ClassID为%d，错误为：%v", appClass.ClassId, err)
		return nil, err
	}

	result := &server.AppCharacteristics{
		AppName:     appName,
		SectionData: make([]*core.SectionData, len(metric.Data)),
	}

	typ := reflect.TypeOf(core.SectionData{})
	for i, datum := range metric.Data {
		classVal := reflect.ValueOf(datum).Elem()
		sectionData := &core.SectionData{}
		appVal := reflect.ValueOf(sectionData).Elem()

		for fi := 0; fi < classVal.NumField(); fi++ {
			field := typ.Field(fi)
			if field.Name[:3] == "Cpu" {
				appVal.FieldByName(field.Name).SetFloat(classVal.FieldByName(field.Name).Float() * float64(appClass.CpuMax))
			} else /*Mem*/ {
				appVal.FieldByName(field.Name).SetFloat(classVal.FieldByName(field.Name).Float() * float64(appClass.MemMax))
			}
		}

		result.SectionData[i] = sectionData
	}

	return result, nil
}

func (s *serverImpl) ReCluster() {
	s.executeReCluster <- struct{}{}
}
