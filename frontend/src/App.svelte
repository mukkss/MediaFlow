<script>
  import { onMount } from 'svelte';

  let activeTab = 'dashboard';
  let isMonitoring = false;
  let config = {
    input_dir: '',
    output_dir: '',
    worker_count: 2,
    video_crf: 20,
    video_preset: 'fast',
    audio_bitrate: '192k',
    jpeg_quality: 92,
    scan_interval_seconds: 30
  };

  let stats = {
    TotalDetected: 0,
    TotalConverted: 0,
    TotalSkipped: 0,
    TotalFailed: 0,
    ActiveWorkers: 0
  };

  let recentJobs = [];
  let saveMessage = '';

  onMount(async () => {
    // If running inside Wails desktop runtime
    if (window.go && window.go.app && window.go.app.AppService) {
      loadInitialState();
      // Listen to Wails events
      if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn('job_update', (job) => {
          updateJob(job);
        });
        window.runtime.EventsOn('stats_update', (newStats) => {
          stats = newStats;
        });
        window.runtime.EventsOn('monitoring_state', (state) => {
          isMonitoring = state;
        });
      }
    } else {
      // Mock / preview mode
      mockPreviewState();
    }
  });

  async function loadInitialState() {
    try {
      config = await window.go.app.AppService.GetConfig();
      isMonitoring = await window.go.app.AppService.IsMonitoring();
      stats = await window.go.app.AppService.GetStats();
      recentJobs = await window.go.app.AppService.GetRecentJobs();
    } catch (e) {
      console.error("Failed to load initial state from Go:", e);
    }
  }

  function mockPreviewState() {
    config.input_dir = 'C:\\Users\\User\\Pictures\\Camera_Import';
    config.output_dir = 'C:\\Users\\User\\Pictures\\Camera_Import\\Converted';
    stats = {
      TotalDetected: 24,
      TotalConverted: 18,
      TotalSkipped: 5,
      TotalFailed: 1,
      ActiveWorkers: 1
    };
    recentJobs = [
      { SourcePath: 'DSC_0042.CR3', DestPath: 'DSC_0042.jpg', Status: 'Completed', Error: '' },
      { SourcePath: 'DSC_0043.MOV', DestPath: 'DSC_0043.mp4', Status: 'Processing', Error: '' },
      { SourcePath: 'DSC_0044.ARW', DestPath: 'DSC_0044.jpg', Status: 'Queued', Error: '' },
      { SourcePath: 'DSC_0040.JPG', DestPath: '', Status: 'Skipped', Error: 'Already compatible JPEG' },
      { SourcePath: 'CORRUPT_01.MP4', DestPath: '', Status: 'Failed', Error: 'Moov atom missing' }
    ];
  }

  function updateJob(job) {
    const idx = recentJobs.findIndex(j => j.SourcePath === job.SourcePath);
    if (idx !== -1) {
      recentJobs[idx] = job;
      recentJobs = [...recentJobs];
    } else {
      recentJobs = [job, ...recentJobs.slice(0, 49)];
    }
  }

  async function toggleMonitoring() {
    if (window.go && window.go.app && window.go.app.AppService) {
      if (isMonitoring) {
        await window.go.app.AppService.Stop();
        isMonitoring = false;
      } else {
        try {
          await window.go.app.AppService.Start();
          isMonitoring = true;
        } catch (err) {
          alert("Error: " + err);
        }
      }
    } else {
      isMonitoring = !isMonitoring;
    }
  }

  async function openConvertedFolder() {
    if (window.go && window.go.app && window.go.app.AppService) {
      await window.go.app.AppService.OpenFolder(config.output_dir);
    } else {
      alert("Opening folder: " + (config.output_dir || config.input_dir + '\\Converted'));
    }
  }

  async function saveSettings() {
    if (window.go && window.go.app && window.go.app.AppService) {
      try {
        await window.go.app.AppService.UpdateConfig(config);
        saveMessage = 'Settings saved successfully!';
        setTimeout(() => saveMessage = '', 3000);
      } catch (e) {
        alert("Failed to save settings: " + e);
      }
    } else {
      saveMessage = 'Settings updated (Preview Mode)';
      setTimeout(() => saveMessage = '', 3000);
    }
  }

  function getFileName(path) {
    if (!path) return '';
    return path.split(/[\/\\]/).pop();
  }
</script>

<div class="container">
  <header>
    <div class="brand">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
        <circle cx="12" cy="13" r="4"/>
      </svg>
      <h1>MediaFlow</h1>
      <span class="status-badge {isMonitoring ? 'status-active' : 'status-idle'}">
        <span class="status-dot"></span>
        {isMonitoring ? 'Monitoring Active' : 'Monitoring Paused'}
      </span>
    </div>
    <nav>
      <button class="{activeTab === 'dashboard' ? 'active' : ''}" on:click={() => activeTab = 'dashboard'}>Dashboard</button>
      <button class="{activeTab === 'settings' ? 'active' : ''}" on:click={() => activeTab = 'settings'}>Settings</button>
      <button class="{activeTab === 'about' ? 'active' : ''}" on:click={() => activeTab = 'about'}>About</button>
    </nav>
  </header>

  <main>
    {#if activeTab === 'dashboard'}
      <div class="action-bar">
        <div class="path-info">
          <div class="path-row">
            <span class="path-label">Import Folder:</span>
            <span class="path-text">{config.input_dir || 'Not configured (set in Settings)'}</span>
          </div>
          <div class="path-row">
            <span class="path-label">Converted To:</span>
            <span class="path-text">{config.output_dir || (config.input_dir ? config.input_dir + '\\Converted' : 'Default (Input/Converted)')}</span>
          </div>
        </div>
        <div class="btn-group">
          <button class="btn btn-secondary" on:click={openConvertedFolder}>
            📂 Open Converted
          </button>
          <button class="btn {isMonitoring ? 'btn-danger' : 'btn-success'}" on:click={toggleMonitoring}>
            {isMonitoring ? '⏹ Stop Monitoring' : '▶ Start Monitoring'}
          </button>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-title">Detected Files</div>
          <div class="stat-value">{stats.TotalDetected}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Converted</div>
          <div class="stat-value" style="color: var(--success);">{stats.TotalConverted}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Skipped (Cached)</div>
          <div class="stat-value" style="color: var(--text-muted);">{stats.TotalSkipped}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Failed / Issues</div>
          <div class="stat-value" style="color: {stats.TotalFailed > 0 ? 'var(--danger)' : 'var(--text-main)'};">{stats.TotalFailed}</div>
        </div>
      </div>

      <div class="activity-section">
        <div class="section-header">
          <span>Live Activity & Processing Queue</span>
          {#if stats.ActiveWorkers > 0}
            <span style="color: #60a5fa; font-size: 12px;">⚡ {stats.ActiveWorkers} Worker(s) Active</span>
          {/if}
        </div>
        <div class="table-container">
          <table>
            <thead>
              <tr>
                <th>File</th>
                <th>Status</th>
                <th>Destination / Notes</th>
              </tr>
            </thead>
            <tbody>
              {#if recentJobs.length === 0}
                <tr>
                  <td colspan="3" style="text-align: center; color: var(--text-muted); padding: 32px;">
                    No media files detected yet. Connect a camera or copy files to the import folder.
                  </td>
                </tr>
              {:else}
                {#each recentJobs as job}
                  <tr>
                    <td style="font-weight: 500;">{getFileName(job.SourcePath)}</td>
                    <td>
                      <span class="tag tag-{job.Status}">{job.Status}</span>
                    </td>
                    <td style="color: var(--text-muted);">
                      {#if job.Status === 'Completed'}
                        {getFileName(job.DestPath)}
                      {:else if job.Status === 'Failed'}
                        <span style="color: var(--danger);">{job.Error}</span>
                      {:else if job.Status === 'Skipped'}
                        <span>{job.Error || 'Already converted'}</span>
                      {:else}
                        <span>Processing in background...</span>
                      {/if}
                    </td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      </div>

    {:else if activeTab === 'settings'}
      <div class="settings-panel">
        <h2 style="font-size: 18px; margin-bottom: 20px;">Configuration Settings</h2>

        {#if saveMessage}
          <div style="background-color: rgba(16, 185, 129, 0.2); color: var(--success); padding: 10px; border-radius: var(--radius); margin-bottom: 16px;">
            {saveMessage}
          </div>
        {/if}

        <div class="form-group">
          <label>Camera Import Folder</label>
          <div class="desc">The folder on your PC or SD card where camera photos and videos land.</div>
          <input type="text" class="form-control" bind:value={config.input_dir} placeholder="C:\Users\...\Pictures\Camera_Import" />
        </div>

        <div class="form-group">
          <label>Converted Output Folder (Optional)</label>
          <div class="desc">Leave blank to default to a 'Converted' subfolder inside the import directory.</div>
          <input type="text" class="form-control" bind:value={config.output_dir} placeholder="Leave empty for Input\Converted" />
        </div>

        <div class="form-group">
          <label>JPEG Image Quality</label>
          <div class="desc">Compression quality for camera RAW, HEIC, and images (Recommended: 92%).</div>
          <div class="range-wrap">
            <input type="range" min="75" max="100" bind:value={config.jpeg_quality} />
            <span class="range-val">{config.jpeg_quality}%</span>
          </div>
        </div>

        <div class="form-group">
          <label>Video H.264 Quality Factor (CRF)</label>
          <div class="desc">Lower means higher visual quality (18-20 is visually lossless, 23 is standard).</div>
          <div class="range-wrap">
            <input type="range" min="16" max="28" bind:value={config.video_crf} />
            <span class="range-val">{config.video_crf}</span>
          </div>
        </div>

        <div class="form-group">
          <label>Concurrent Workers</label>
          <div class="desc">Number of files converted in parallel (Default: 2).</div>
          <div class="range-wrap">
            <input type="range" min="1" max="8" bind:value={config.worker_count} />
            <span class="range-val">{config.worker_count}</span>
          </div>
        </div>

        <div style="margin-top: 28px; display: flex; justify-content: flex-end;">
          <button class="btn btn-primary" on:click={saveSettings}>
            Save Configuration
          </button>
        </div>
      </div>

    {:else if activeTab === 'about'}
      <div class="settings-panel">
        <h2 style="font-size: 18px; margin-bottom: 12px;">About MediaFlow</h2>
        <p style="color: var(--text-muted); line-height: 1.6; margin-bottom: 16px;">
          MediaFlow is an automatic, completely offline Windows desktop utility that monitors your camera imports and develops them into phone-compatible formats.
        </p>
        <ul style="color: var(--text-muted); line-height: 1.8; margin-left: 20px; margin-bottom: 24px;">
          <li><strong>Videos</strong>: Transcoded to Faststart H.264 MP4 with AAC audio</li>
          <li><strong>Camera RAW</strong>: Developed to high-quality JPEG with EXIF preservation</li>
          <li><strong>HEIC / 10-bit HIF</strong>: Converted to JPEG for seamless sharing</li>
          <li><strong>Originals</strong>: Always left untouched</li>
        </ul>
        <div style="font-size: 12px; color: #64748b;">
          MediaFlow v1.0.0 • Go & Wails Desktop Runtime
        </div>
      </div>
    {/if}
  </main>
</div>
