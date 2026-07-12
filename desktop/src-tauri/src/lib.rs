use std::sync::Mutex;
use tauri::{AppHandle, Manager, Emitter};

struct SidecarState {
    _child: Mutex<Option<std::process::Child>>,
}

fn find_sidecar_binary() -> Result<std::path::PathBuf, String> {
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

fn spawn_backend(app: &AppHandle) -> Result<(), String> {
    let data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("failed to resolve data dir: {e}"))?;
    std::fs::create_dir_all(&data_dir)
        .map_err(|e| format!("failed to create data dir: {e}"))?;

    let binary = find_sidecar_binary()?;
    eprintln!("[sidecar] using binary: {}", binary.display());

    let child = std::process::Command::new(&binary)
        .env("TEAMS_ADDR", "127.0.0.1:7801")
        .env("TEAMS_DB_PATH", data_dir.join("teams.db").to_string_lossy().as_ref())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .spawn()
        .map_err(|e| format!("failed to spawn backend: {e}"))?;

    app.manage(SidecarState {
        _child: Mutex::new(Some(child)),
    });

    // Notify frontend when backend is ready
    let app_handle = app.clone();
    std::thread::spawn(move || {
        std::thread::sleep(std::time::Duration::from_millis(1500));
        let _ = app_handle.emit("backend-ready", ());
    });

    eprintln!("[sidecar] backend spawned successfully on 127.0.0.1:7801");
    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            if let Err(e) = spawn_backend(&app.handle()) {
                eprintln!("WARNING: backend sidecar failed to start: {e}");
                eprintln!("The app will run without backend API. Start it manually if needed.");
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
