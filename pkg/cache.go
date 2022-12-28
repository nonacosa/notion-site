package pkg

type NotionCache struct {
	ParentPropInfo  *NotionProp
	ParentFilesInfo *Files
	ChildDatabaseId string
}

type NotionCaches struct {
	TODO []*NotionCache
}

func NewNotionCaches() []*NotionCache {
	return []*NotionCache{}

}

func (caches *NotionCaches) SetCache(blockId string) {

}
