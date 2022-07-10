package stripe

type ProductModel struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}
