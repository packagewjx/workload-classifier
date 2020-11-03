# Kubernetes负载分类服务器

## 下载与构建

需事先准备好go环境

```shell
mkdir -p $GOPATH/src/github.com/packagewjx/
cd $GOPATH/src/github.com/packagewjx
git clone git@github.com:packagewjx/workload-classifier.git
go install github.com/packagewjx/workload-classifier
```

## 使用方法

运行`workload-classifier --help`可以获取帮助信息。

```
$ ./workload-classifier --help                                                 
负载分类器，拥有对alibaba_trace v2018进行处理，以及获取集群最新数据并更新分类的功能

Usage:
  workload-classifier [command]

Available Commands:
  cluster     读取数据文件聚类计算，并输出结果到新文件中
  help        Help about any command
  preprocess  预处理相关指令
  server      负载分类服务器

Flags:
      --config string   config file (default is $HOME/.workload-classifier.yaml)
  -h, --help            help for workload-classifier
  -t, --toggle          Help message for toggle

Use "workload-classifier [command] --help" for more information about a command.
```

### cluster命令

```
$ ./workload-classifier cluster --help
读取数据文件聚类计算，并输出结果到新文件中

Usage:
  workload-classifier cluster dataFile outputFile numClass [flags]

Flags:
  -a, --algorithm string      指定使用的算法。默认为kmeans，可选值：kmeans (default "kmeans")
  -f, --dataFormat string     数据文件格式
  -h, --help                  help for cluster
      --kMeansRound int       K-Means算法执行的轮次 (default 30)
  -p, --outputPrecision int   输出文件数据精度，默认为2 (default 2)
  -r, --removeColumn ints     需要移除的列号，从0开始计算。使用此字段忽略掉不是数字的列

Global Flags:
      --config string   config file (default is $HOME/.workload-classifier.yaml)
```

cluster命令是手动聚类使用的命令。设计上本命令能用于任意类型的输入，而不仅仅是负载的特征。本命令支持读取csv类型的文件，计算`numClass`个聚类后，输出到`outputFile`中。

### preprocess命令

```
$ ./workload-classifier preprocess --help
预处理相关指令

Usage:
  workload-classifier preprocess [command]

Available Commands:
  convert     将container_usage格式的文件转换为特征，并输出到文件中
  impute      填充NaN数据
  normalize   将转换后的数据标准化
  split       根据AppDU将container_usage.csv分割成小文件

Flags:
  -h, --help   help for preprocess

Global Flags:
      --config string   config file (default is $HOME/.workload-classifier.yaml)

Use "workload-classifier preprocess [command] --help" for more information about a command.
```

本命令用于预处理从其他数据源获取的文件。目前支持的数据源是[阿里巴巴v2018](https://github.com/alibaba/clusterdata/tree/master/cluster-trace-v2018)数据源。

split指令用于将原本长度为100多GB的大文件，依照appdu属性，分割成一个个小文件，便于在小内存的服务器上进行处理。

convert指令用于将原始数据转换为本系统所使用的特征。

impute是数据填充指令。目前采用线性方式填充数据，以数据的两端为端点。因为数据集中不可避免的会出现缺失的数据，因为应用不再运行或者出错结束，或者应用重启等情况。

normalize则是用于将数据标准化。因为负载的运行特征均不太相同，使用的资源数量都不同，我们更加关注一个负载的资源使用变化而不是具体使用多少资源，因此需要将所有的数据进行标准化之后再聚类。

### server命令

```
$ ./workload-classifier server --help    
本服务器将从Kubernetes的metrics server中每隔一段时间（通过interval指定）获取一次监控数据，并保存记录。
监控数据仅保留最近一段时间（通过duration设置）。服务器每天定时（通过re-cluster-time设置）聚类，更新分类数据，
以确保数据反映近期的真实情况。用户可以通过本服务器提供的接口获取应用属于哪个类别的数据。

Usage:
  workload-classifier server [flags]

Flags:
  -f, --center-file string         初始中心文件。若不为空，则启动时将会读取此文件并作为各个类别的数据，此过程将删除旧有数据。若为空，则使用原类数据
  -c, --class uint                 聚类类别数量 (default 20)
  -d, --duration duration          保存数据的时间，至少为1天 (default 168h0m0s)
  -h, --help                       help for server
  -i, --interval duration          获取监控数据的间隔，至少为15s (default 1m0s)
      --mysql-host string          Mysql服务器主机端口，格式为：host:port。若为空，则读取环境变量MYSQL_SERVICE_HOST与MYSQL_SERVICE_PORT取得
  -p, --port uint16                服务端口号 (default 2000)
  -t, --re-cluster-time duration   每天定时跑聚类算法的时间，值应该小于24小时 (default 1h0m0s)
  -r, --round uint                 聚类迭代次数 (default 30)

Global Flags:
      --config string   config file (default is $HOME/.workload-classifier.yaml)
```

## API

#### /namespaces/${名称空间}/appcharacteristics/${应用名称}

用于获取一个应用程序的一天内的运行特征。

应用名称说明：应用名称对应的是Kubernetes内的`ReplicaSet`、`DaemonSet`、`StatefulSet`的数据，而不是Pod的名称，因为Pod是单独部署的，每个Pod都有独一无二的名称，获取一个Pod的运行特征对部署新的Pod无参考意义，因为名称不同，无法得知是否是同一个应用。因此使用Pod的Owner，也就是上述三者的名称作为应用名称。

返回值定义在`pkg/server/types.go`文件中，类型如下：

```go
type AppCharacteristics struct {
	AppName `json:",inline"`

	SectionData []*core.SectionData `json:"sectionData"`
}

type AppName struct {
	Name      string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
	Namespace string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
}
```

`SectionData`定义在`pkg/core/types.go`文件中，如下：

```go
type SectionData struct {
	CpuAvg float32 `json:"cpuAvg"`
	CpuMax float32 `json:"cpuMax"`
	CpuMin float32 `json:"cpuMin"`
	CpuP50 float32 `json:"cpuP50"`
	CpuP90 float32 `json:"cpuP90"`
	CpuP99 float32 `json:"cpuP99"`
	MemAvg float32 `json:"memAvg"`
	MemMax float32 `json:"memMax"`
	MemMin float32 `json:"memMin"`
	MemP50 float32 `json:"memP50"`
	MemP90 float32 `json:"memP90"`
	MemP99 float32 `json:"memP99"`
}
```

#### /recluster

本API不带任何参数，指定服务器进行重新聚类的操作。

#### /healthz

本API不带任何参数，用于确认服务器是否正常在运行。

## Docker容器构建

`Makefile`中定义了用于构建Docker镜像的命令。主要目标的用途如下

- docker-scheduler：构建支持负载感知的调度算法插件的Kubernetes调度器。
- docker-workload-classifier：构建负载分类服务器镜像。
- docker-push：将构建好的镜像推送到docker hub中

## Kubernetes部署

`deploy.yaml`中是将本负载分类服务器部署到Kubernetes上的命令。在可用的Kubernetes集群中，使用如下命令使用

```shell
kubectl apply -f deploy.yaml
```

命令运行完毕后，在所有镜像拉取下来并运行之后，负载分类服务器将运行，使用下面命令查询其情况

```
kubectl get deployments -n workload-classifier
```