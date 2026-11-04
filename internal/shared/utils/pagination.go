package utils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type StateQueryParams struct {
	Pagination
	State uint64 `form:"state"`
	Query string `form:"search"`
}

type ManagerQueryParams struct {
	Pagination
	Query string `form:"name"`
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func ParsePagination(c *gin.Context) (Pagination, error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return Pagination{}, errors.New("parámetro 'page' inválido: debe ser un número entero positivo")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		return Pagination{}, errors.New("parámetro 'page_size' inválido: debe ser un número entero positivo")
	}

	return Pagination{Page: page, PageSize: pageSize}, nil
}

func ParseStateQuery(c *gin.Context) (StateQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return StateQueryParams{}, err
	}

	state, err := parseUint64(c.DefaultQuery("state", "0"))
	if err != nil {
		return StateQueryParams{}, errors.New("parámetro 'state' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return StateQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return StateQueryParams{
		Pagination: p,
		State:      state,
		Query:      query,
	}, nil
}

func ParseManagerQuery(c *gin.Context) (ManagerQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return ManagerQueryParams{}, err
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return ManagerQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return ManagerQueryParams{
		Pagination: p,
		Query:      query,
	}, nil
}
