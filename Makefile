DOCKER_WORKLOAD_TAG = packagewjx/workload-classifier:latest
DOCKER_SCHEDULER_TAG = packagewjx/feature-aware-scheduler:latest

.PHONY: all
all: docker-scheduler docker-workload-classifier

workload-classifier:
	CGO_ENABLED=0 go build -a -ldflags '-s' -o workload-classifier .

scheduler:
	CGO_ENABLED=0 go build -a -ldflags '-s' -o scheduler ./cmd/scheduler

.PHONY: docker-scheduler
docker-scheduler: scheduler
	cp scheduler docker/scheduler
	docker build -t ${DOCKER_SCHEDULER_TAG} docker/scheduler
	rm docker/scheduler/scheduler

.PHONY: docker-workload-classifier
docker-workload-classifier: workload-classifier
	cp workload-classifier docker/workload-classifier
	docker build -t ${DOCKER_WORKLOAD_TAG} docker/workload-classifier
	rm docker/workload-classifier/workload-classifier

.PHONY: docker-push
docker-push: docker-scheduler docker-workload-classifier
	docker push ${DOCKER_WORKLOAD_TAG}
	docker push ${DOCKER_SCHEDULER_TAG}