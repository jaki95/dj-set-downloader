# Bug Fix Summary: Index Out of Bounds and Ignored FileExtension

## Issues Fixed

### 1. Index Out of Bounds Error
**Problem**: The application panicked with "index out of bounds" when audio files lacked file extensions. This occurred because `filepath.Ext()` returns an empty string for files without extensions, and attempting to slice `[1:]` on an empty string caused a runtime error.

**Affected locations**:
- `internal/server/handlers.go:137`
- `internal/server/download_handlers.go:116`
- `internal/server/helpers.go:116`

**Solution**: Created a safe `getFileExtension()` helper method that:
- Checks if `filepath.Ext()` returns an empty string
- Defaults to "mp3" if no extension is found
- Safely extracts the extension without the dot prefix

### 2. Ignored User-Requested File Extension
**Problem**: The system always used the extension of the downloaded input file, ignoring the user's requested output file extension from `req.FileExtension`.

**Affected location**: 
- `internal/server/handlers.go` - `process()` function

**Solution**: 
- Modified the `process()` function to accept a `requestedExtension` parameter
- Added logic to use the user's requested extension when provided
- Falls back to the input file's extension if no specific extension is requested

## Changes Made

### 1. Added Safe Extension Helper
```go
// getFileExtension safely extracts the file extension without causing index out of bounds
func (s *Server) getFileExtension(filePath string) string {
    ext := filepath.Ext(filePath)
    if ext == "" {
        return "mp3" // Default extension if none found
    }
    return strings.ToLower(ext[1:]) // Remove the dot and convert to lowercase
}
```

### 2. Updated Process Function
- Added `requestedExtension string` parameter to `process()` function
- Added logic to prioritize user-requested extension over input file extension
- Updated function call in `processUrlInBackground()` to pass `req.FileExtension`

### 3. Replaced Unsafe Extension Extraction
- Replaced all instances of `strings.ToLower(filepath.Ext(filePath)[1:])` with `s.getFileExtension(filePath)`
- Removed unused imports from `download_handlers.go`

## Result
- ✅ Application no longer panics when processing files without extensions
- ✅ User-requested file extensions are now respected
- ✅ All code compiles successfully
- ✅ Maintains backward compatibility with existing functionality