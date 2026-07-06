package request

type ProductListRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	Keyword  string `json:"keyword"`
	Category string `json:"category"`
	Status   *int   `json:"status"`
}

func (r ProductListRequest) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

func (r ProductListRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 20
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

func (r ProductListRequest) Offset() int {
	return (r.GetPage() - 1) * r.GetLimit()
}

type SaveProductRequest struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	PriceMin           int64  `json:"priceMin"`
	PriceMax           int64  `json:"priceMax"`
	SellingPoints      string `json:"sellingPoints"`
	SuitablePeople     string `json:"suitablePeople"`
	UnsuitablePeople   string `json:"unsuitablePeople"`
	Scenarios          string `json:"scenarios"`
	Specs              string `json:"specs"`
	IndustryAttributes string `json:"industryAttributes"`
	ImageURL           string `json:"imageUrl"`
	Priority           int    `json:"priority"`
	KnowledgeBaseID    int64  `json:"knowledgeBaseId"`
	Status             int    `json:"status"`
	Remark             string `json:"remark"`
}

type DeleteProductRequest struct {
	ID int64 `json:"id"`
}

type UpdateProductStatusRequest struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type ReindexProductRequest struct {
	ID int64 `json:"id"`
}
