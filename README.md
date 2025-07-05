# DJ Set Downloader

This application provides an API to download DJ sets and split them into individual tracks based on a provided SoundCloud URL and tracklist. Files are processed and stored temporarily with automatic cleanup.

## Features

- Download DJ sets from SoundCloud
- Split audio into individual tracks with metadata
- Download for individual or all tracks
- Temporary file storage with automatic cleanup
- Real-time job progress tracking

## API Endpoints

### Job Management
- `POST /api/process` - Start processing a DJ set
- `GET /api/jobs` - List all jobs (paginated)
- `GET /api/jobs/{id}` - Get job status and download URLs
- `POST /api/jobs/{id}/cancel` - Cancel a running job

### Download Endpoints
- `GET /api/jobs/{id}/download` - Download all tracks as ZIP
- `GET /api/jobs/{id}/tracks` - Get track metadata and download URLs
- `GET /api/jobs/{id}/tracks/{trackNumber}/download` - Download single track

### Health Check
- `GET /health` - Server health status

## Usage

1. **Start the server:**

   ```bash
   docker compose up
   ```

2. **Submit a processing job:**

   ```bash
   curl -X POST http://localhost:8000/api/process \
   -H "Content-Type: application/json" \
   -d '{
     "url": "SOUNDCLOUD_URL_OF_THE_SET",
     "tracklist": "{\"artist\":\"Artist Name\",\"name\":\"Mix Name\",\"tracks\":[{\"artist\":\"Artist 1\",\"name\":\"Track 1\",\"start_time\":\"00:00\",\"end_time\":\"03:45\"},{\"artist\":\"Artist 2\",\"name\":\"Track 2\",\"start_time\":\"03:45\",\"end_time\":\"07:30\"}]}",
     "fileExtension": "mp3"
   }'
   ```

   Response:
   ```json
   {
     "message": "Processing started",
     "jobId": "123"
   }
   ```

3. **Check job status:**

   ```bash
   curl http://localhost:8000/api/jobs/123
   ```

   Response (when completed):
   ```json
   {
     "id": "123",
     "status": "completed",
     "progress": 100,
     "message": "Processing completed successfully",
     "tracks": [
       {
         "track_number": 1,
         "name": "Track 1",
         "artist": "Artist 1",
         "download_url": "/api/jobs/123/tracks/1/download",
         "size_bytes": 5242880,
         "available": true
       }
     ],
     "download_all_url": "/api/jobs/123/download",
     "total_tracks": 2
   }
   ```

4. **Download tracks:**

   ```bash
   # Download all tracks as ZIP
   curl -O -J http://localhost:8000/api/jobs/123/download
   
   # Download individual track
   curl -O -J http://localhost:8000/api/jobs/123/tracks/1/download
   ```

## Request Format

### Process Request
```json
{
  "url": "https://soundcloud.com/artist/set-name",
  "tracklist": "{\"artist\":\"Artist Name\",\"name\":\"Mix Name\",\"tracks\":[...]}",
  "fileExtension": "mp3",          // Optional: mp3, m4a, wav, flac
  "maxConcurrentTasks": 4          // Optional: 1-10
}
```

### Tracklist Format
```json
{
  "artist": "DJ Name",
  "name": "Mix Title", 
  "year": 2024,
  "genre": "Electronic",
  "tracks": [
    {
      "name": "Track Name",
      "artist": "Track Artist",
      "start_time": "00:00:00",    // Format: HH:MM:SS or MM:SS
      "end_time": "03:45:00"       // Optional for last track
    }
  ]
}
```

## File Management

- **Temporary Storage**: Files are stored in system temp directory under `djset-server-jobs/`
- **Automatic Cleanup**: Files older than 24 hours are automatically removed
- **No Permanent Storage**: Files are not permanently stored on disk
- **Per-Job Directories**: Each job gets its own isolated directory

## Configuration

Edit `config/config.yaml`:
```yaml
log_level: -4                    # Debug level
file_extension: m4a             # Default output format

storage:
  type: "local"
  output_dir: "output"          # Legacy - now uses temp storage
```
