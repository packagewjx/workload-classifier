package server

type API interface {
	QueryAppClass(appName AppName) (*ClassMetrics, error)

	ReCluster()
}

func (s *serverImpl) QueryAppClass(appName AppName) (*ClassMetrics, error) {
	s.logger.Printf("接收到查询名称空间为%s，名称为%s的请求\n", appName.Namespace, appName.Name)
	classId, err := s.dao.QueryAppClassIdByApp(&appName)
	if err == ErrAppNotFound {
		return nil, err
	} else if err != nil {
		s.logger.Printf("查询AppClass失败，原因为：%v\n", err)
		return nil, err
	}

	metric, err := s.dao.QueryClassMetricsByClassId(classId)
	if err != nil {
		s.logger.Printf("查询ClassMetrics时出错，ClassID为%d，错误为：%v", classId, err)
		return nil, err
	}

	return metric, nil
}

func (s *serverImpl) ReCluster() {
	s.executeReCluster <- struct{}{}
}
