package service

import (
	"ecommerce/services/product/internal/model"
	"ecommerce/services/product/internal/repository"
)

type ProductService interface {
	CreateProduct(req model.CreateProductRequest) (*model.Product, error)
	GetAllProducts() ([]model.Product, error)
	GetProductByID(id uint) (*model.Product, error)
}

type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) CreateProduct(req model.CreateProductRequest) (*model.Product, error) {
	product := model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}
	err := s.repo.Create(&product)
	return &product, err
}

func (s *productService) GetAllProducts() ([]model.Product, error) {
	return s.repo.FindAll()
}

func (s *productService) GetProductByID(id uint) (*model.Product, error) {
	return s.repo.FindById(id)
}
