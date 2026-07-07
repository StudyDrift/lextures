package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- shared types ---

type fileFolder struct {
	ID        string  `json:"id"`
	CourseID  string  `json:"courseId"`
	ParentID  *string `json:"parentId"`
	Name      string  `json:"name"`
	CreatedBy *string `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type fileItem struct {
	ID               string  `json:"id"`
	CourseID         string  `json:"courseId"`
	FolderID         *string `json:"folderId"`
	StorageKey       string  `json:"storageKey"`
	OriginalFilename string  `json:"originalFilename"`
	DisplayName      string  `json:"displayName"`
	MimeType         string  `json:"mimeType"`
	ByteSize         int64   `json:"byteSize"`
	UploadedBy       *string `json:"uploadedBy"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type folderContents struct {
	FolderID *string      `json:"folderId"`
	Folders  []fileFolder `json:"folders"`
	Files    []fileItem   `json:"files"`
}

type uploadInitResponse struct {
	ObjectKey       string  `json:"objectKey"`
	PresignedPutURL *string `json:"presignedPutUrl"`
	ExpiresAt       *string `json:"expiresAt"`
	CourseID        string  `json:"courseId"`
	FolderID        *string `json:"folderId"`
	// Also present when local storage returns the committed item directly:
	ID               *string `json:"id"`
	OriginalFilename *string `json:"originalFilename"`
	DisplayName      *string `json:"displayName"`
	MimeType         *string `json:"mimeType"`
	ByteSize         *int64  `json:"byteSize"`
}

// --- root command ---

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage course files and folders",
}

// --- files list ---

var filesListFlags struct {
	course string
	folder string
}

var filesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files and folders in a course (or inside a folder)",
	RunE:  runFilesList,
}

var filesUsageFlags struct {
	course string
}

var filesUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show storage usage vs quota for a course",
	RunE:  runFilesUsage,
}

func runFilesUsage(cmd *cobra.Command, _ []string) error {
	usage, raw, err := fetchCourseStorageUsage(client.New(Cfg.Server, Cfg.APIKey), filesUsageFlags.course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	limit := "unlimited"
	if usage.LimitBytes != nil {
		limit = formatFileBytes(*usage.LimitBytes)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "course %s: used %s / %s (%.1f%%)\n",
		filesUsageFlags.course, formatFileBytes(usage.UsedBytes), limit, usage.PercentUsed)
	return nil
}

func init() {
	filesUsageCmd.Flags().StringVar(&filesUsageFlags.course, "course", "", "course code (required)")
	_ = filesUsageCmd.MarkFlagRequired("course")

	filesListCmd.Flags().StringVar(&filesListFlags.course, "course", "", "course code (required)")
	_ = filesListCmd.MarkFlagRequired("course")
	filesListCmd.Flags().StringVar(&filesListFlags.folder, "folder", "", "folder UUID (omit for root)")
}

func runFilesList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)

	var path string
	if filesListFlags.folder != "" {
		path = "/api/v1/courses/" + url.PathEscape(filesListFlags.course) +
			"/files/folders/" + url.PathEscape(filesListFlags.folder)
	} else {
		path = "/api/v1/courses/" + url.PathEscape(filesListFlags.course) + "/files"
	}

	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var body folderContents
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(body)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tID\tNAME\tSIZE\tUPDATED")
	for _, f := range body.Folders {
		_, _ = fmt.Fprintf(w, "folder\t%s\t%s\t-\t%s\n", f.ID, f.Name, f.UpdatedAt)
	}
	for _, fi := range body.Files {
		_, _ = fmt.Fprintf(w, "file\t%s\t%s\t%s\t%s\n",
			fi.ID, fi.DisplayName, formatFileBytes(fi.ByteSize), fi.UpdatedAt)
	}
	return w.Flush()
}

// --- files mkdir ---

var filesMkdirFlags struct {
	course string
	name   string
	parent string
}

var filesMkdirCmd = &cobra.Command{
	Use:   "mkdir",
	Short: "Create a folder in a course",
	RunE:  runFilesMkdir,
}

func init() {
	filesMkdirCmd.Flags().StringVar(&filesMkdirFlags.course, "course", "", "course code (required)")
	_ = filesMkdirCmd.MarkFlagRequired("course")
	filesMkdirCmd.Flags().StringVar(&filesMkdirFlags.name, "name", "", "folder name (required)")
	_ = filesMkdirCmd.MarkFlagRequired("name")
	filesMkdirCmd.Flags().StringVar(&filesMkdirFlags.parent, "parent", "", "parent folder UUID (omit for root)")
}

func runFilesMkdir(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)

	payload := map[string]any{"name": filesMkdirFlags.name}
	if filesMkdirFlags.parent != "" {
		payload["parentId"] = filesMkdirFlags.parent
	} else {
		payload["parentId"] = nil
	}

	raw, _ := json.Marshal(payload)
	req, err := c.NewRequest(http.MethodPost,
		"/api/v1/courses/"+url.PathEscape(filesMkdirFlags.course)+"/files/folders",
		bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var folder fileFolder
	if err := json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(folder)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created folder %s (%s)\n", folder.Name, folder.ID)
	return nil
}

// --- files upload ---

var filesUploadFlags struct {
	course string
	folder string
	quiet  bool
}

var filesUploadCmd = &cobra.Command{
	Use:   "upload <local-path>",
	Short: "Upload a local file to a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runFilesUpload,
}

func init() {
	filesUploadCmd.Flags().StringVar(&filesUploadFlags.course, "course", "", "course code (required)")
	_ = filesUploadCmd.MarkFlagRequired("course")
	filesUploadCmd.Flags().StringVar(&filesUploadFlags.folder, "folder", "", "destination folder UUID (omit for root)")
	filesUploadCmd.Flags().BoolVar(&filesUploadFlags.quiet, "quiet", false, "suppress progress output")
}

// filesProgressOut allows tests to override the progress writer.
var filesProgressOut io.Writer

func runFilesUpload(cmd *cobra.Command, args []string) error {
	localPath := filepath.Clean(args[0])
	if strings.Contains(localPath, "..") {
		return fmt.Errorf("invalid file path: %s", args[0])
	}

	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", localPath)
		}
		return fmt.Errorf("accessing file %s: %w", localPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", localPath)
	}
	fileSize := info.Size()

	progressOut := filesProgressOut
	if progressOut == nil {
		progressOut = cmd.OutOrStdout()
	}

	if !filesUploadFlags.quiet {
		_, _ = fmt.Fprintf(progressOut, "Uploading %s (%s)...\n",
			filepath.Base(localPath), formatFileBytes(fileSize))
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()

	mimeType := mime.TypeByExtension(filepath.Ext(localPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	c := client.New(Cfg.Server, Cfg.APIKey)

	params := url.Values{"filename": {filepath.Base(localPath)}}
	if filesUploadFlags.folder != "" {
		params.Set("folderId", filesUploadFlags.folder)
	}
	initPath := "/api/v1/courses/" + url.PathEscape(filesUploadFlags.course) +
		"/files/items?" + params.Encode()

	var bodyReader io.Reader = f
	if !filesUploadFlags.quiet && fileSize > 1024*1024 {
		bodyReader = &fileProgressReader{r: f, total: fileSize, out: progressOut}
	}

	initReq, err := c.NewRequest(http.MethodPost, initPath, bodyReader)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	initReq.Header.Set("Content-Type", mimeType)

	initResp, err := c.Do(initReq)
	if err != nil {
		return fmt.Errorf("initiating upload: %w", err)
	}
	defer func() { _ = initResp.Body.Close() }()

	if !filesUploadFlags.quiet && fileSize > 1024*1024 {
		_, _ = fmt.Fprintln(progressOut)
	}

	if initResp.StatusCode != http.StatusCreated && initResp.StatusCode != http.StatusOK {
		return apiError(initResp, 2)
	}

	initBody, err := io.ReadAll(initResp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var initData uploadInitResponse
	if err := json.Unmarshal(initBody, &initData); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	// If the server returned a presigned URL we need to PUT the file there and then confirm.
	if initData.PresignedPutURL != nil && *initData.PresignedPutURL != "" {
		if !filesUploadFlags.quiet {
			_, _ = fmt.Fprintln(progressOut, "Uploading to storage...")
		}

		f2, err := os.Open(localPath)
		if err != nil {
			return fmt.Errorf("reopening file for presigned upload: %w", err)
		}
		defer func() { _ = f2.Close() }()

		putReq, err := http.NewRequest(http.MethodPut, *initData.PresignedPutURL, f2)
		if err != nil {
			return fmt.Errorf("building presigned PUT: %w", err)
		}
		putReq.Header.Set("Content-Type", mimeType)
		putReq.ContentLength = fileSize

		putResp, err := http.DefaultClient.Do(putReq)
		if err != nil {
			return fmt.Errorf("uploading to storage: %w", err)
		}
		_ = putResp.Body.Close()
		if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("storage upload failed (HTTP %d)", putResp.StatusCode)
		}

		// Confirm with the server.
		confirmPayload := map[string]any{
			"objectKey": initData.ObjectKey,
			"filename":  filepath.Base(localPath),
			"mimeType":  mimeType,
			"byteSize":  fileSize,
		}
		if filesUploadFlags.folder != "" {
			confirmPayload["folderId"] = filesUploadFlags.folder
		} else {
			confirmPayload["folderId"] = nil
		}

		confirmRaw, _ := json.Marshal(confirmPayload)
		confirmPath := "/api/v1/courses/" + url.PathEscape(filesUploadFlags.course) +
			"/files/items/confirm"
		confirmReq, err := c.NewRequest(http.MethodPost, confirmPath, bytes.NewReader(confirmRaw))
		if err != nil {
			return fmt.Errorf("building confirm request: %w", err)
		}

		confirmResp, err := doWithRetry(c, confirmReq)
		if err != nil {
			return fmt.Errorf("confirming upload: %w", err)
		}
		defer func() { _ = confirmResp.Body.Close() }()

		if confirmResp.StatusCode != http.StatusCreated && confirmResp.StatusCode != http.StatusOK {
			return apiError(confirmResp, 2)
		}

		var item fileItem
		if err := json.NewDecoder(confirmResp.Body).Decode(&item); err != nil {
			return fmt.Errorf("decoding confirm response: %w", err)
		}
		return printUploadedFile(cmd, item)
	}

	// Local storage path: server already committed the file and returned a FileItem.
	if initData.ID != nil {
		item := fileItem{
			ID:               *initData.ID,
			CourseID:         initData.CourseID,
			StorageKey:       initData.ObjectKey,
			OriginalFilename: filepath.Base(localPath),
			DisplayName:      filepath.Base(localPath),
			MimeType:         mimeType,
		}
		if initData.OriginalFilename != nil {
			item.OriginalFilename = *initData.OriginalFilename
		}
		if initData.DisplayName != nil {
			item.DisplayName = *initData.DisplayName
		}
		if initData.MimeType != nil {
			item.MimeType = *initData.MimeType
		}
		if initData.ByteSize != nil {
			item.ByteSize = *initData.ByteSize
		}
		if initData.FolderID != nil {
			item.FolderID = initData.FolderID
		}
		return printUploadedFile(cmd, item)
	}

	// Fallback: print raw JSON.
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(initBody)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Upload complete.")
	return nil
}

func printUploadedFile(cmd *cobra.Command, item fileItem) error {
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Uploaded %s (%s)\n",
		item.DisplayName, item.ID)
	return nil
}

// fileProgressReader prints upload progress while streaming.
type fileProgressReader struct {
	r     io.Reader
	total int64
	read  int64
	out   io.Writer
}

func (p *fileProgressReader) Read(buf []byte) (n int, err error) {
	n, err = p.r.Read(buf)
	p.read += int64(n)
	if p.total > 0 {
		pct := p.read * 100 / p.total
		_, _ = fmt.Fprintf(p.out, "\rUploading... %3d%%", pct)
	}
	return
}

// --- files download ---

var filesDownloadFlags struct {
	course string
	out    string
}

var filesDownloadCmd = &cobra.Command{
	Use:   "download <item-id>",
	Short: "Download a file from a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runFilesDownload,
}

func init() {
	filesDownloadCmd.Flags().StringVar(&filesDownloadFlags.course, "course", "", "course code (required)")
	_ = filesDownloadCmd.MarkFlagRequired("course")
	filesDownloadCmd.Flags().StringVar(&filesDownloadFlags.out, "out", "", "output path (default: item's filename in current directory)")
}

func runFilesDownload(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)

	contentPath := "/api/v1/courses/" + url.PathEscape(filesDownloadFlags.course) +
		"/files/items/" + url.PathEscape(itemID) + "/content"

	req, err := c.NewRequest(http.MethodGet, contentPath, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	// Determine output path.
	outPath := filesDownloadFlags.out
	if outPath == "" {
		// Try to derive filename from Content-Disposition header.
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			_, params, err := mime.ParseMediaType(cd)
			if err == nil {
				if name, ok := params["filename"]; ok && name != "" {
					outPath = name
				}
			}
		}
		if outPath == "" {
			outPath = itemID
		}
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"path":  outPath,
			"bytes": written,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %s (%s)\n", outPath, formatFileBytes(written))
	return nil
}

// --- files rename ---

var filesRenameFlags struct {
	course string
	item   string
	folder string
}

var filesRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename a file (--item) or folder (--folder)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFilesRename,
}

func init() {
	filesRenameCmd.Flags().StringVar(&filesRenameFlags.course, "course", "", "course code (required)")
	_ = filesRenameCmd.MarkFlagRequired("course")
	filesRenameCmd.Flags().StringVar(&filesRenameFlags.item, "item", "", "file item UUID (mutually exclusive with --folder)")
	filesRenameCmd.Flags().StringVar(&filesRenameFlags.folder, "folder", "", "folder UUID (mutually exclusive with --item)")
}

func runFilesRename(cmd *cobra.Command, args []string) error {
	newName := args[0]
	if filesRenameFlags.item == "" && filesRenameFlags.folder == "" {
		return fmt.Errorf("one of --item or --folder is required")
	}
	if filesRenameFlags.item != "" && filesRenameFlags.folder != "" {
		return fmt.Errorf("--item and --folder are mutually exclusive")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)

	if filesRenameFlags.item != "" {
		raw, _ := json.Marshal(map[string]string{"displayName": newName})
		req, err := c.NewRequest(http.MethodPatch,
			"/api/v1/courses/"+url.PathEscape(filesRenameFlags.course)+
				"/files/items/"+url.PathEscape(filesRenameFlags.item),
			bytes.NewReader(raw))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}

		resp, err := doWithRetry(c, req)
		if err != nil {
			return fmt.Errorf("renaming file: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return apiError(resp, 2)
		}

		var item fileItem
		if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Renamed file to %s\n", item.DisplayName)
		return nil
	}

	// Folder rename.
	raw, _ := json.Marshal(map[string]string{"name": newName})
	req, err := c.NewRequest(http.MethodPatch,
		"/api/v1/courses/"+url.PathEscape(filesRenameFlags.course)+
			"/files/folders/"+url.PathEscape(filesRenameFlags.folder),
		bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("renaming folder: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var folder fileFolder
	if err := json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(folder)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Renamed folder to %s\n", folder.Name)
	return nil
}

// --- files move ---

var filesMoveFlags struct {
	course string
	item   string
	folder string
	to     string
}

var filesMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move a file (--item) or folder (--folder) to a new parent folder (--to; omit for root)",
	RunE:  runFilesMove,
}

func init() {
	filesMoveCmd.Flags().StringVar(&filesMoveFlags.course, "course", "", "course code (required)")
	_ = filesMoveCmd.MarkFlagRequired("course")
	filesMoveCmd.Flags().StringVar(&filesMoveFlags.item, "item", "", "file item UUID (mutually exclusive with --folder)")
	filesMoveCmd.Flags().StringVar(&filesMoveFlags.folder, "folder", "", "folder UUID to move (mutually exclusive with --item)")
	filesMoveCmd.Flags().StringVar(&filesMoveFlags.to, "to", "", "destination folder UUID (omit to move to root)")
}

func runFilesMove(cmd *cobra.Command, _ []string) error {
	if filesMoveFlags.item == "" && filesMoveFlags.folder == "" {
		return fmt.Errorf("one of --item or --folder is required")
	}
	if filesMoveFlags.item != "" && filesMoveFlags.folder != "" {
		return fmt.Errorf("--item and --folder are mutually exclusive")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)

	if filesMoveFlags.item != "" {
		// Move file: PATCH with folderId (empty string = root).
		folderID := filesMoveFlags.to
		raw, _ := json.Marshal(map[string]string{"folderId": folderID})
		req, err := c.NewRequest(http.MethodPatch,
			"/api/v1/courses/"+url.PathEscape(filesMoveFlags.course)+
				"/files/items/"+url.PathEscape(filesMoveFlags.item),
			bytes.NewReader(raw))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}

		resp, err := doWithRetry(c, req)
		if err != nil {
			return fmt.Errorf("moving file: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return apiError(resp, 2)
		}

		var item fileItem
		if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
		}
		dest := "root"
		if item.FolderID != nil && *item.FolderID != "" {
			dest = *item.FolderID
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Moved %s to %s\n", item.DisplayName, dest)
		return nil
	}

	// Move folder: PATCH with parentId (empty string = root).
	parentID := filesMoveFlags.to
	raw, _ := json.Marshal(map[string]string{"parentId": parentID})
	req, err := c.NewRequest(http.MethodPatch,
		"/api/v1/courses/"+url.PathEscape(filesMoveFlags.course)+
			"/files/folders/"+url.PathEscape(filesMoveFlags.folder),
		bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("moving folder: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var folder fileFolder
	if err := json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(folder)
	}
	dest := "root"
	if folder.ParentID != nil && *folder.ParentID != "" {
		dest = *folder.ParentID
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Moved folder %s to %s\n", folder.Name, dest)
	return nil
}

// --- files delete ---

var filesDeleteFlags struct {
	course string
	item   string
	folder string
	force  bool
}

var filesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a file (--item) or folder (--folder)",
	RunE:  runFilesDelete,
}

func init() {
	filesDeleteCmd.Flags().StringVar(&filesDeleteFlags.course, "course", "", "course code (required)")
	_ = filesDeleteCmd.MarkFlagRequired("course")
	filesDeleteCmd.Flags().StringVar(&filesDeleteFlags.item, "item", "", "file item UUID (mutually exclusive with --folder)")
	filesDeleteCmd.Flags().StringVar(&filesDeleteFlags.folder, "folder", "", "folder UUID (mutually exclusive with --item)")
	filesDeleteCmd.Flags().BoolVar(&filesDeleteFlags.force, "force", false, "skip confirmation prompt")
}

// filesDeleteInput allows tests to inject a reader for the confirmation prompt.
var filesDeleteInput io.Reader

func runFilesDelete(cmd *cobra.Command, _ []string) error {
	if filesDeleteFlags.item == "" && filesDeleteFlags.folder == "" {
		return fmt.Errorf("one of --item or --folder is required")
	}
	if filesDeleteFlags.item != "" && filesDeleteFlags.folder != "" {
		return fmt.Errorf("--item and --folder are mutually exclusive")
	}

	isFile := filesDeleteFlags.item != ""
	id := filesDeleteFlags.item
	kind := "file"
	if !isFile {
		id = filesDeleteFlags.folder
		kind = "folder"
	}

	if !filesDeleteFlags.force {
		in := filesDeleteInput
		if in == nil {
			in = os.Stdin
		}
		r := bufio.NewReader(in)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete %s %q? (y/N) ", kind, id)
		line, _ := r.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	c := client.New(Cfg.Server, Cfg.APIKey)

	var path string
	if isFile {
		path = "/api/v1/courses/" + url.PathEscape(filesDeleteFlags.course) +
			"/files/items/" + url.PathEscape(id)
	} else {
		path = "/api/v1/courses/" + url.PathEscape(filesDeleteFlags.course) +
			"/files/folders/" + url.PathEscape(id)
	}

	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", kind, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiError(resp, 2)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"deleted": id, "type": kind,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s %s\n", kind, id)
	return nil
}

// --- helpers ---

func formatFileBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func init() {
	filesCmd.AddCommand(
		filesListCmd,
		filesMkdirCmd,
		filesUploadCmd,
		filesDownloadCmd,
		filesRenameCmd,
		filesMoveCmd,
		filesDeleteCmd,
		filesUsageCmd,
	)
	rootCmd.AddCommand(filesCmd)
}

