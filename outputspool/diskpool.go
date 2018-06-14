package outputspool


type DiskStrayOutputsPool struct {
	cfg *PoolConfig
}


func NewDiskStrayOutputsPool(config *PoolConfig) StrayOutputsPool {
	return &DiskStrayOutputsPool{
		cfg: config,
	}
}