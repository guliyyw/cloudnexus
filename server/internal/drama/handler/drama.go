package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudnexus/server/internal/drama/service"
	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var taskWebSocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

type DramaHandler struct {
	svc *service.DramaService
}

func NewDramaHandler(svc *service.DramaService) *DramaHandler {
	return &DramaHandler{svc: svc}
}

func (h *DramaHandler) HandleListProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.svc.ListProjects(httputil.GetUserID(c), c.Query("keyword"), c.DefaultQuery("sort", "updated_desc"), page, pageSize)
	if err != nil {
		c.JSON(500, response.Error(500, "查询项目失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"items": items, "total": total, "page": page, "page_size": pageSize}))
}

func (h *DramaHandler) HandleCreateProject(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	project, err := h.svc.CreateProject(httputil.GetUserID(c), req.Title, req.Description)
	if err != nil {
		c.JSON(500, response.Error(500, "创建项目失败"))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"project": project}))
}

func (h *DramaHandler) HandleGetProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	detail, err := h.svc.GetProject(httputil.GetUserID(c), id)
	if err != nil {
		c.JSON(404, response.Error(404, "项目不存在"))
		return
	}
	c.JSON(200, response.OKWithData(detail))
}

func (h *DramaHandler) HandleUpdateProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Preface     string `json:"preface"`
		Settings    string `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	project, err := h.svc.UpdateProject(httputil.GetUserID(c), id, req.Title, req.Description, req.Preface, req.Settings)
	if err != nil {
		c.JSON(500, response.Error(500, "保存项目失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"project": project}))
}

func (h *DramaHandler) HandleDeleteProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteProject(httputil.GetUserID(c), id); err != nil {
		c.JSON(500, response.Error(500, "删除项目失败"))
		return
	}
	c.JSON(200, response.OK("删除成功"))
}

func (h *DramaHandler) HandleParseScript(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Script string `json:"script"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	detail, err := h.svc.ParseAndSave(httputil.GetUserID(c), id, req.Script)
	if err != nil {
		c.JSON(500, response.Error(500, "解析剧本失败"))
		return
	}
	c.JSON(200, response.OKWithData(detail))
}

func (h *DramaHandler) HandleUpdateStoryboard(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	storyboardID, ok := parseID(c, "storyboardId")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content"`
		Prompt  string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	storyboard, err := h.svc.UpdateStoryboard(httputil.GetUserID(c), projectID, storyboardID, req.Content, req.Prompt)
	if err != nil {
		c.JSON(500, response.Error(500, "保存分镜失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"storyboard": storyboard}))
}

func (h *DramaHandler) HandleSelectStoryboardMedia(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	storyboardID, ok := parseID(c, "storyboardId")
	if !ok {
		return
	}
	mediaID, ok := parseID(c, "mediaId")
	if !ok {
		return
	}
	storyboard, err := h.svc.SelectStoryboardMedia(httputil.GetUserID(c), projectID, storyboardID, mediaID)
	if err != nil {
		c.JSON(500, response.Error(500, "选择分镜媒体失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"storyboard": storyboard}))
}

func (h *DramaHandler) HandleDeleteStoryboardMedia(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	storyboardID, ok := parseID(c, "storyboardId")
	if !ok {
		return
	}
	mediaID, ok := parseID(c, "mediaId")
	if !ok {
		return
	}
	storyboard, media, err := h.svc.DeleteStoryboardMedia(httputil.GetUserID(c), projectID, storyboardID, mediaID)
	if err != nil {
		c.JSON(500, response.Error(500, "删除分镜媒体失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"storyboard": storyboard, "media": media}))
}

func (h *DramaHandler) HandleImportStoryboardSegments(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	storyboardID, ok := parseID(c, "storyboardId")
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(400, response.Error(400, "请粘贴片段分析结果"))
		return
	}
	segments, err := h.svc.ImportStoryboardSegments(httputil.GetUserID(c), projectID, storyboardID, req.Text)
	if err != nil {
		c.JSON(400, response.Error(400, "片段分析结果格式不正确"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"segments": segments}))
}

func (h *DramaHandler) HandleAppendStoryboards(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Suffix string `json:"suffix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Suffix == "" {
		c.JSON(400, response.Error(400, "请输入追加内容"))
		return
	}
	storyboards, err := h.svc.AppendToStoryboards(httputil.GetUserID(c), projectID, req.Suffix)
	if err != nil {
		c.JSON(500, response.Error(500, "批量追加失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"storyboards": storyboards}))
}

func (h *DramaHandler) HandleUpdateAsset(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	assetID, ok := parseID(c, "assetId")
	if !ok {
		return
	}
	var req struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		ReferencePrompt string `json:"reference_prompt"`
		VoiceName       string `json:"voice_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	asset, err := h.svc.UpdateAsset(httputil.GetUserID(c), projectID, assetID, req.Name, req.Description, req.ReferencePrompt, req.VoiceName)
	if err != nil {
		c.JSON(500, response.Error(500, "保存资产失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"asset": asset}))
}

func (h *DramaHandler) HandleImportAssets(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(400, response.Error(400, "请粘贴 AI 解析结果"))
		return
	}
	assets, err := h.svc.ImportAssetsFromAI(httputil.GetUserID(c), projectID, req.Text)
	if err != nil {
		c.JSON(500, response.Error(500, "导入资产失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"assets": assets}))
}

func (h *DramaHandler) HandleUploadAssetReference(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	assetID, ok := parseID(c, "assetId")
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, response.Error(400, "请选择参考图"))
		return
	}
	asset, err := h.svc.UploadAssetReference(httputil.GetUserID(c), projectID, assetID, file)
	if err != nil {
		c.JSON(500, response.Error(500, "上传参考图失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"asset": asset}))
}

func (h *DramaHandler) HandleUploadStoryboardAudio(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	storyboardID, ok := parseID(c, "storyboardId")
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, response.Error(400, "请选择音频文件"))
		return
	}
	durationMS, _ := strconv.Atoi(c.DefaultPostForm("duration_ms", "0"))
	storyboard, err := h.svc.UploadStoryboardAudio(httputil.GetUserID(c), projectID, storyboardID, file, durationMS)
	if err != nil {
		c.JSON(500, response.Error(500, "上传音频失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"storyboard": storyboard}))
}

func (h *DramaHandler) HandleBatchImportAudio(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, response.Error(400, "请选择音频文件"))
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(400, response.Error(400, "请选择音频文件"))
		return
	}
	results, err := h.svc.BatchImportAudio(httputil.GetUserID(c), projectID, files)
	if err != nil {
		c.JSON(500, response.Error(500, "导入音频失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"results": results}))
}

func (h *DramaHandler) HandleExportProject(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	saveToDrive := c.DefaultQuery("save", "true") != "false"
	data, filename, err := h.svc.ExportProject(httputil.GetUserID(c), projectID, saveToDrive)
	if err != nil {
		c.JSON(500, response.Error(500, "导出项目失败"))
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

func (h *DramaHandler) HandleImportProject(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, response.Error(400, "请选择项目文件"))
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(500, response.Error(500, "读取项目文件失败"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(500, response.Error(500, "读取项目文件失败"))
		return
	}
	project, err := h.svc.ImportProject(httputil.GetUserID(c), data)
	if err != nil {
		c.JSON(400, response.Error(400, "项目文件格式不正确"))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"project": project}))
}

func (h *DramaHandler) HandleListTasks(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	detail, err := h.svc.GetProject(httputil.GetUserID(c), projectID)
	if err != nil {
		c.JSON(404, response.Error(404, "项目不存在"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"items": detail.Tasks}))
}

func (h *DramaHandler) HandleCreateTask(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req service.CreateTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	task, err := h.svc.CreateTask(httputil.GetUserID(c), projectID, req)
	if err != nil {
		c.JSON(500, response.Error(500, "创建任务失败"))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"task": task}))
}

func (h *DramaHandler) HandleCancelTask(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	taskID, ok := parseID(c, "taskId")
	if !ok {
		return
	}
	task, err := h.svc.CancelTask(httputil.GetUserID(c), projectID, taskID)
	if err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"task": task}))
}

func (h *DramaHandler) HandleRetryTask(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	taskID, ok := parseID(c, "taskId")
	if !ok {
		return
	}
	task, err := h.svc.RetryTask(httputil.GetUserID(c), projectID, taskID)
	if err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"task": task}))
}

func (h *DramaHandler) HandleTaskWebSocket(c *gin.Context) {
	events, unsubscribe, err := h.svc.SubscribeTaskEvents(httputil.GetUserID(c))
	if err != nil {
		c.JSON(503, response.Error(503, err.Error()))
		return
	}
	defer unsubscribe()
	conn, err := taskWebSocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case event := <-events:
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (h *DramaHandler) HandleGetSetting(c *gin.Context) {
	setting, err := h.svc.GetSetting(httputil.GetUserID(c))
	if err != nil {
		c.JSON(500, response.Error(500, "读取设置失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"setting": setting}))
}

func (h *DramaHandler) HandleComfyStatus(c *gin.Context) {
	status, err := h.svc.GetComfyStatus(c.Request.Context(), httputil.GetUserID(c), c.Query("url"))
	if err != nil {
		c.JSON(500, response.Error(500, "读取 ComfyUI 状态失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"status": status}))
}

func (h *DramaHandler) HandleSaveSetting(c *gin.Context) {
	var req model.DramaSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	setting, err := h.svc.SaveSetting(httputil.GetUserID(c), req)
	if err != nil {
		c.JSON(500, response.Error(500, "保存设置失败"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"setting": setting}))
}

func parseID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效 ID"))
		return 0, false
	}
	return id, true
}
