package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"dash/backend/internal/assets"
	"dash/backend/internal/config"
	"dash/backend/internal/services"
	"dash/backend/internal/store"
	"dash/backend/internal/widgets"
)

type API struct {
	cfg     config.Config
	log     *zap.Logger
	store   *store.Store
	assets  *assets.Service
	widgets *widgets.Registry
}

func NewRouter(cfg config.Config, log *zap.Logger, st *store.Store) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(log))
	router.Use(jsonBodyLimit(1 << 20))
	if len(cfg.AllowedOrigins) > 0 {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.AllowedOrigins,
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
			AllowCredentials: false,
		}))
	}

	api := &API{
		cfg:     cfg,
		log:     log,
		store:   st,
		assets:  assets.New(cfg.IconDir(), cfg.PublicBaseURL, cfg.MaxIconUploadBytes, st),
		widgets: widgets.NewRegistry(st, cfg.WidgetFetchTimeout, cfg.WidgetCacheTTL),
	}

	router.StaticFS("/uploads/icons", http.Dir(cfg.IconDir()))
	router.GET("/health", api.health)

	v1 := router.Group("/api/v1")
	v1.GET("/dashboard", api.dashboard)
	v1.GET("/groups", api.listGroups)
	v1.POST("/groups", api.createGroup)
	v1.GET("/groups/:groupId", api.getGroup)
	v1.PATCH("/groups/:groupId", api.patchGroup)
	v1.DELETE("/groups/:groupId", api.deleteGroup)

	v1.GET("/services", api.listServices)
	v1.POST("/services", api.createService)
	v1.GET("/services/:serviceId", api.getService)
	v1.PATCH("/services/:serviceId", api.patchService)
	v1.DELETE("/services/:serviceId", api.deleteService)

	v1.PUT("/layout", api.putLayout)
	v1.POST("/assets/icons", api.uploadIcon)

	v1.GET("/widgets", api.listWidgets)
	v1.POST("/widgets", api.createWidget)
	v1.GET("/widgets/:widgetId", api.getWidget)
	v1.PATCH("/widgets/:widgetId", api.patchWidget)
	v1.DELETE("/widgets/:widgetId", api.deleteWidget)
	v1.GET("/widgets/:widgetId/data", api.widgetData)
	v1.POST("/widgets/:widgetId/refresh", api.refreshWidget)

	return router
}

func jsonBodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.ContentType() == "application/json" {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func (a *API) health(c *gin.Context) {
	if err := a.store.Ping(c.Request.Context()); err != nil {
		a.error(c, http.StatusServiceUnavailable, "internal_error", "database unavailable", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) dashboard(c *gin.Context) {
	dashboard, err := a.store.Dashboard(c.Request.Context())
	if err != nil {
		a.internal(c, err)
		return
	}
	dashboard.Widgets = services.MaskWidgets(dashboard.Widgets)
	c.JSON(http.StatusOK, dashboard)
}

func (a *API) listGroups(c *gin.Context) {
	groups, err := a.store.Groups(c.Request.Context())
	if err != nil {
		a.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (a *API) createGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if !bindJSON(c, &req, a) {
		return
	}
	name := req.Name
	input, err := services.NormalizeGroup(store.GroupInput{Name: &name}, true)
	if err != nil {
		a.validation(c, err)
		return
	}
	group, err := a.store.CreateGroup(c.Request.Context(), *input.Name)
	if err != nil {
		a.internal(c, err)
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (a *API) getGroup(c *gin.Context) {
	group, err := a.store.Group(c.Request.Context(), c.Param("groupId"))
	a.respondOne(c, group, err)
}

func (a *API) patchGroup(c *gin.Context) {
	var input store.GroupInput
	if !bindJSON(c, &input, a) {
		return
	}
	normalized, err := services.NormalizeGroup(input, false)
	if err != nil {
		a.validation(c, err)
		return
	}
	group, err := a.store.UpdateGroup(c.Request.Context(), c.Param("groupId"), normalized)
	a.respondOne(c, group, err)
}

func (a *API) deleteGroup(c *gin.Context) {
	move := c.Query("moveServicesToUngrouped") == "true"
	err := a.store.DeleteGroup(c.Request.Context(), c.Param("groupId"), move)
	a.respondNoContent(c, err)
}

func (a *API) listServices(c *gin.Context) {
	services, err := a.store.Services(c.Request.Context())
	if err != nil {
		a.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, services)
}

func (a *API) createService(c *gin.Context) {
	var input store.ServiceInput
	if !bindJSON(c, &input, a) {
		return
	}
	normalized, err := services.NormalizeService(input)
	if err != nil {
		a.validation(c, err)
		return
	}
	service, err := a.store.CreateService(c.Request.Context(), normalized)
	a.respondCreated(c, service, err)
}

func (a *API) getService(c *gin.Context) {
	service, err := a.store.Service(c.Request.Context(), c.Param("serviceId"))
	a.respondOne(c, service, err)
}

func (a *API) patchService(c *gin.Context) {
	var input store.ServiceInput
	if !bindJSON(c, &input, a) {
		return
	}
	normalized, err := services.NormalizeService(input)
	if err != nil {
		a.validation(c, err)
		return
	}
	service, err := a.store.UpdateService(c.Request.Context(), c.Param("serviceId"), normalized)
	a.respondOne(c, service, err)
}

func (a *API) deleteService(c *gin.Context) {
	err := a.store.DeleteService(c.Request.Context(), c.Param("serviceId"))
	a.respondNoContent(c, err)
}

func (a *API) putLayout(c *gin.Context) {
	var input store.LayoutInput
	if !bindJSON(c, &input, a) {
		return
	}
	dashboard, err := a.store.ApplyLayout(c.Request.Context(), input)
	if err != nil {
		a.respondErr(c, err)
		return
	}
	dashboard.Widgets = services.MaskWidgets(dashboard.Widgets)
	c.JSON(http.StatusOK, dashboard)
}

func (a *API) uploadIcon(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, a.cfg.MaxIconUploadBytes+1024)
	file, err := c.FormFile("file")
	if err != nil {
		a.validation(c, err)
		return
	}
	assetFile, err := a.assets.SaveIcon(c.Request, file)
	if err != nil {
		switch {
		case errors.Is(err, assets.ErrTooLarge):
			a.error(c, http.StatusRequestEntityTooLarge, "upload_too_large", "icon upload exceeds max size", nil)
		case errors.Is(err, assets.ErrUnsupportedMedia):
			a.error(c, http.StatusUnsupportedMediaType, "unsupported_media_type", "icon must be PNG, JPEG, WebP, or SVG", nil)
		default:
			a.internal(c, err)
		}
		return
	}
	c.JSON(http.StatusCreated, assetFile)
}

func (a *API) listWidgets(c *gin.Context) {
	widgets, err := a.store.Widgets(c.Request.Context())
	if err != nil {
		a.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, services.MaskWidgets(widgets))
}

func (a *API) createWidget(c *gin.Context) {
	var input store.WidgetInput
	if !bindJSON(c, &input, a) {
		return
	}
	normalized, err := services.NormalizeWidget(input, true)
	if err != nil {
		a.validation(c, err)
		return
	}
	widget, err := a.store.CreateWidget(c.Request.Context(), normalized)
	if err == nil {
		widget = services.MaskWidget(widget)
	}
	a.respondCreated(c, widget, err)
}

func (a *API) getWidget(c *gin.Context) {
	widget, err := a.store.Widget(c.Request.Context(), c.Param("widgetId"))
	if err == nil {
		widget = services.MaskWidget(widget)
	}
	a.respondOne(c, widget, err)
}

func (a *API) patchWidget(c *gin.Context) {
	var input store.WidgetInput
	if !bindJSON(c, &input, a) {
		return
	}
	current, err := a.store.Widget(c.Request.Context(), c.Param("widgetId"))
	if err != nil {
		a.respondErr(c, err)
		return
	}
	normalized, err := services.NormalizeWidgetPatch(current, input)
	if err != nil {
		a.validation(c, err)
		return
	}
	widget, err := a.store.UpdateWidget(c.Request.Context(), c.Param("widgetId"), normalized)
	if err == nil {
		widget = services.MaskWidget(widget)
	}
	a.respondOne(c, widget, err)
}

func (a *API) deleteWidget(c *gin.Context) {
	err := a.store.DeleteWidget(c.Request.Context(), c.Param("widgetId"))
	a.respondNoContent(c, err)
}

func (a *API) widgetData(c *gin.Context) {
	widget, err := a.store.Widget(c.Request.Context(), c.Param("widgetId"))
	if err != nil {
		a.log.Warn("widget lookup failed", zap.String("widgetId", c.Param("widgetId")), zap.Error(err))
		a.respondErr(c, err)
		return
	}
	data, err := a.widgets.Data(c.Request.Context(), widget)
	if err != nil {
		a.log.Warn("widget data fetch failed", zap.String("widgetId", widget.ID), zap.String("type", widget.Type), zap.Error(err))
	}
	a.respondOne(c, data, err)
}

func (a *API) refreshWidget(c *gin.Context) {
	widget, err := a.store.Widget(c.Request.Context(), c.Param("widgetId"))
	if err != nil {
		a.respondErr(c, err)
		return
	}
	data, err := a.widgets.Refresh(c.Request.Context(), widget)
	a.respondOne(c, data, err)
}

func (a *API) respondCreated(c *gin.Context, value any, err error) {
	if err != nil {
		a.respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}

func (a *API) respondOne(c *gin.Context, value any, err error) {
	if err != nil {
		a.respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (a *API) respondNoContent(c *gin.Context, err error) {
	if err != nil {
		a.respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		a.error(c, http.StatusNotFound, "not_found", "resource not found", nil)
	case errors.Is(err, store.ErrConflict):
		a.error(c, http.StatusConflict, "conflict", "operation conflicts with current state", nil)
	case errors.Is(err, store.ErrValidation):
		a.error(c, http.StatusBadRequest, "validation_error", err.Error(), nil)
	case isPostgresValidationError(err):
		a.error(c, http.StatusBadRequest, "validation_error", "request references invalid data", nil)
	default:
		a.internal(c, err)
	}
}

func isPostgresValidationError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "22P02", "23502", "23503", "23514":
		return true
	default:
		return false
	}
}

func (a *API) validation(c *gin.Context, err error) {
	a.error(c, http.StatusBadRequest, "validation_error", err.Error(), nil)
}

func (a *API) internal(c *gin.Context, err error) {
	a.log.Error("request failed", zap.Error(err))
	a.error(c, http.StatusInternalServerError, "internal_error", "internal server error", nil)
}

func (a *API) error(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, ErrorResponse{Code: code, Message: message, Details: details})
}

func bindJSON(c *gin.Context, dst any, a *API) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		a.validation(c, err)
		return false
	}
	return true
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func requestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		status := c.Writer.Status()
		if status >= 500 {
			log.Error("http request",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", status),
				zap.String("bytes", strconv.Itoa(c.Writer.Size())),
			)
			return
		}
		log.Debug("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
		)
	}
}
