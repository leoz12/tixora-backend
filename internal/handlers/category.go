package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/services"
)

// CategoryHandler handles event category endpoints.
type CategoryHandler struct {
	categoryService services.ICategoryService
}

func NewCategoryHandler(categoryService services.ICategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// ListCategories handles GET /api/categories
//
//	@Summary		List categories
//	@Description	Returns the list of event categories. Inactive categories are excluded unless include_inactive=true.
//	@Tags			categories
//	@Produce		json
//	@Param			include_inactive	query		bool	false	"Include inactive categories"
//	@Success		200					{object}	dto.SuccessResponse{data=[]dto.CategoryResponse}
//	@Failure		500					{object}	dto.ErrorResponse
//	@Router			/categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	includeInactive, _ := strconv.ParseBool(c.Query("include_inactive"))

	categories, err := h.categoryService.ListCategories(c.Request.Context(), includeInactive)
	if err != nil {
		respondError(c, err, "Failed to fetch categories")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewCategoryResponseList(categories)))
}

// GetCategoryByID handles GET /api/categories/:id
//
//	@Summary		Get category by ID
//	@Description	Returns a single category by its ID.
//	@Tags			categories
//	@Produce		json
//	@Param			id	path		string	true	"Category ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	id := c.Param("id")

	category, err := h.categoryService.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err, "Failed to fetch category")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewCategoryResponse(category)))
}

// CreateCategory handles POST /api/categories (admin only)
//
//	@Summary		Create category
//	@Description	Creates a new event category. Requires admin authentication.
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.CategoryRequest	true	"Category details"
//	@Success		201		{object}	dto.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	category, err := h.categoryService.CreateCategory(c.Request.Context(), req.Name)
	if err != nil {
		respondError(c, err, "Failed to create category")
		return
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse("Category created", dto.NewCategoryResponse(category)))
}

// UpdateCategory handles PUT /api/categories/:id (admin only)
//
//	@Summary		Update category
//	@Description	Updates an existing event category. Requires admin authentication.
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		string					true	"Category ID"
//	@Param			request	body		dto.CategoryRequest	true	"Category details"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.CategoryResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var req dto.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	category, err := h.categoryService.UpdateCategory(c.Request.Context(), id, req.Name, req.IsActive)
	if err != nil {
		respondError(c, err, "Failed to update category")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Category updated", dto.NewCategoryResponse(category)))
}

// DeleteCategory handles DELETE /api/categories/:id (admin only)
//
//	@Summary		Delete category
//	@Description	Deletes an event category. Fails if the category is still used by existing events. Requires admin authentication.
//	@Tags			categories
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Category ID"
//	@Success		200	{object}	dto.SuccessResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	if err := h.categoryService.DeleteCategory(c.Request.Context(), id); err != nil {
		respondError(c, err, "Failed to delete category")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Category deleted", nil))
}
