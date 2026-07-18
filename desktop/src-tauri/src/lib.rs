use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use tauri::{AppHandle, Emitter, Manager};

const HELP_URL: &str = "https://github.com/danqing-ai/DanQing-Teams#macos-%E5%AE%89%E8%A3%85";

struct SidecarState {
    _child: Mutex<Option<std::process::Child>>,
}

/// Unified user data root: ~/.dq-teams (same as server/cli/tui).
fn teams_home(app: &AppHandle) -> Result<PathBuf, String> {
    let home = app
        .path()
        .home_dir()
        .map_err(|e| format!("failed to resolve home dir: {e}"))?;
    Ok(home.join(".dq-teams"))
}

fn find_sidecar_binary() -> Result<PathBuf, String> {
    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .ok_or_else(|| "cannot determine exe directory".to_string())?;

    let triples = [
        "aarch64-apple-darwin",
        "x86_64-apple-darwin",
        "x86_64-unknown-linux-gnu",
        "x86_64-pc-windows-msvc",
    ];
    for triple in &triples {
        let candidate = exe_dir.join(format!("danqing-teams-backend-{triple}"));
        if candidate.exists() {
            return Ok(candidate);
        }
    }
    Err(format!("sidecar binary not found in {}", exe_dir.display()))
}

/// Copy sidecar out of the .app bundle into ~/.dq-teams/bin.
/// macOS App Translocation / Gatekeeper often kills helpers launched from a
/// quarantined or translocated bundle path; running from the home data dir is stable.
fn prepare_runtime_binary(bundled: &Path, home: &Path) -> Result<PathBuf, String> {
    let bin_dir = home.join("bin");
    fs::create_dir_all(&bin_dir).map_err(|e| format!("create bin dir: {e}"))?;
    let runtime_bin = bin_dir.join("danqing-teams-backend");
    let need_copy = match (fs::metadata(bundled), fs::metadata(&runtime_bin)) {
        (Ok(src), Ok(dst)) => {
            src.len() != dst.len()
                || src
                    .modified()
                    .ok()
                    .zip(dst.modified().ok())
                    .map(|(a, b)| a > b)
                    .unwrap_or(true)
        }
        (Ok(_), Err(_)) => true,
        (Err(e), _) => return Err(format!("sidecar metadata: {e}")),
    };
    if need_copy {
        fs::copy(bundled, &runtime_bin).map_err(|e| format!("copy sidecar: {e}"))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = fs::metadata(&runtime_bin)
                .map_err(|e| format!("sidecar chmod stat: {e}"))?
                .permissions();
            perms.set_mode(0o755);
            fs::set_permissions(&runtime_bin, perms)
                .map_err(|e| format!("sidecar chmod: {e}"))?;
        }
    }
    Ok(runtime_bin)
}

fn spawn_backend(app: &AppHandle) -> Result<(), String> {
    let home = teams_home(app)?;
    fs::create_dir_all(&home).map_err(|e| format!("failed to create ~/.dq-teams: {e}"))?;
    let work_dir = home.join("data");
    fs::create_dir_all(&work_dir).map_err(|e| format!("failed to create data dir: {e}"))?;

    let config_path = home.join("config.yaml");
    let log_path = home.join("backend.log");

    let bundled = find_sidecar_binary()?;
    let binary = prepare_runtime_binary(&bundled, &home)?;
    eprintln!("[sidecar] home: {}", home.display());
    eprintln!("[sidecar] runtime: {}", binary.display());

    let log_file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(&log_path)
        .map_err(|e| format!("open backend log: {e}"))?;
    let log_err = log_file
        .try_clone()
        .map_err(|e| format!("clone backend log: {e}"))?;
    if let Ok(mut f) = OpenOptions::new().create(true).append(true).open(&log_path) {
        let _ = writeln!(f, "\n--- sidecar spawn ---");
    }

    let mut child = std::process::Command::new(&binary)
        .current_dir(&home)
        .env("TEAMS_ADDR", "127.0.0.1:7801")
        .env(
            "TEAMS_DB_PATH",
            home.join("teams.db").to_string_lossy().as_ref(),
        )
        .env("TEAMS_CONFIG", config_path.to_string_lossy().as_ref())
        .env("TEAMS_DATA_DIR", work_dir.to_string_lossy().as_ref())
        .stdout(std::process::Stdio::from(log_file))
        .stderr(std::process::Stdio::from(log_err))
        .spawn()
        .map_err(|e| format!("failed to spawn backend: {e}"))?;

    // Fail fast if the process exits immediately (common under App Translocation).
    std::thread::sleep(Duration::from_millis(400));
    match child.try_wait() {
        Ok(Some(status)) => {
            let tail = fs::read_to_string(&log_path).unwrap_or_default();
            let tail = tail.chars().rev().take(2000).collect::<String>();
            let tail: String = tail.chars().rev().collect();
            return Err(format!(
                "backend exited immediately ({status}). log tail:\n{tail}"
            ));
        }
        Ok(None) => {}
        Err(e) => return Err(format!("backend wait failed: {e}")),
    }

    app.manage(SidecarState {
        _child: Mutex::new(Some(child)),
    });

    let app_handle = app.clone();
    std::thread::spawn(move || {
        std::thread::sleep(Duration::from_millis(1500));
        let _ = app_handle.emit("backend-ready", ());
    });

    eprintln!(
        "[sidecar] backend spawned on 127.0.0.1:7801 (log: {})",
        log_path.display()
    );
    Ok(())
}

/// Open help documentation on first launch (macOS only, due to unsigned app)
fn handle_first_launch(app: &AppHandle) {
    #[cfg(target_os = "macos")]
    {
        if let Ok(home) = teams_home(app) {
            let marker = home.join(".first_launch_done");
            if !marker.exists() {
                let _ = fs::create_dir_all(&home);
                let _ = fs::write(&marker, "1");
                let _ = open::that(HELP_URL);
            }
        }
    }
}

#[tauri::command]
fn open_external(url: String) -> Result<(), String> {
    open::that(&url).map_err(|e| format!("failed to open: {e}"))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![open_external])
        .setup(|app| {
            handle_first_launch(&app.handle());
            if let Err(e) = spawn_backend(&app.handle()) {
                eprintln!("WARNING: backend sidecar failed to start: {e}");
                eprintln!("The app will run without backend API. Start it manually if needed.");
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
