package context

type PipelineDto struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type NfPrDto struct {
	NfName string `json:"nfName"`
	Pr     int    `json:"pr"`
}

type TaskDto struct {
	Id         uint64        `json:"id"`
	Username   string        `json:"username"`
	Status     string        `json:"status"`
	CreateTime int64         `json:"createTime"`
	Pipelines  []PipelineDto `json:"pipelines"`
	NfPrList   []NfPrDto     `json:"nfPrList"`
}
