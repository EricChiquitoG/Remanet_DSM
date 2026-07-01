package src

import (
	"fmt"
	"net/http"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
)

// GET /products
func GetProductsHandler(c *gin.Context) {
	products, err := execdb.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

// GET /products/:id
func GetProductHandler(c *gin.Context) {
	id := c.Param("id")
	product, err := execdb.GetProductByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("product %s not found", id)})
		return
	}
	c.JSON(http.StatusOK, product)
}

// POST /products
func CreateProductHandler(c *gin.Context) {
	var body execdb.ProductCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate component statuses when provided.
	if body.Components != nil {
		fields := map[string]string{
			"bearings": body.Components.Bearings,
			"stator":   body.Components.Stator,
			"rotor":    body.Components.Rotor,
			"housing":  body.Components.Housing,
			"windings": body.Components.Windings,
			"shaft":    body.Components.Shaft,
		}
		for field, val := range fields {
			if !execdb.IsValidComponentStatus(val) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid status %q for component %q: must be Good, Worn, Degraded, or Failed", val, field),
				})
				return
			}
		}
	}

	product, err := execdb.CreateProduct(body)
	if err != nil {
		// Duplicate ID is a 409.
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, product)
}

// GET /products/:id/history
func GetProductHistoryHandler(c *gin.Context) {
	id := c.Param("id")
	record, err := execdb.GetProductHistory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("no history found for product %s", id)})
		return
	}
	c.JSON(http.StatusOK, record)
}
