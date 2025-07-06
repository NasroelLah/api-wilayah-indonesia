import requests
import json
import os
import signal
import sys
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Lock, Event
from tqdm import tqdm
from datetime import datetime

# Global variables for graceful shutdown and thread safety
scraper_state = {
    "current_data": None,
    "checkpoint_file": None,
    "temp_file": None,
    "is_running": False,
    "shutdown_event": Event(),
}

# Thread safety locks
data_lock = Lock()
checkpoint_lock = Lock()
progress_lock = Lock()

# Threading configuration
MAX_WORKERS = 4  # Adjust based on API rate limits and system capacity


def signal_handler(signum, frame):
    """Handle Ctrl+C and other termination signals"""
    if not scraper_state["is_running"]:
        print("\n⚠️ Script sedang tidak berjalan, keluar...")
        sys.exit(0)

    print("\n🛑 Mendeteksi Ctrl+C, menghentikan threads dan menyimpan checkpoint...")

    # Set shutdown event to stop all threads gracefully
    scraper_state["shutdown_event"].set()

    try:
        if scraper_state["current_data"] and scraper_state["checkpoint_file"]:
            safe_checkpoint_save(
                scraper_state["current_data"],
                scraper_state["checkpoint_file"],
                "Disimpan karena script dihentikan paksa",
            )

        if scraper_state["current_data"] and scraper_state["temp_file"]:
            save_to_file(scraper_state["current_data"], scraper_state["temp_file"])

        print("💾 Checkpoint berhasil disimpan!")
        print("🔄 Jalankan ulang script untuk melanjutkan dari posisi terakhir")

    except Exception as e:
        print(f"❌ Error saat menyimpan checkpoint: {e}")

    finally:
        scraper_state["is_running"] = False
        print("👋 Script dihentikan dengan aman")
        sys.exit(0)


# Setup signal handlers
signal.signal(signal.SIGINT, signal_handler)  # Ctrl+C
if hasattr(signal, "SIGTERM"):
    signal.signal(signal.SIGTERM, signal_handler)  # Termination signal


BASE_URL = "https://sipedas.pertanian.go.id/api/wilayah/"
THN = datetime.now().year

# Output folder
os.makedirs("output", exist_ok=True)
os.makedirs("output/checkpoints", exist_ok=True)


def get_json(endpoint, params=None):
    try:
        resp = requests.get(BASE_URL + endpoint, params=params, timeout=10)
        resp.raise_for_status()
        data = resp.json()

        # Clean the received data to handle encoding issues
        if isinstance(data, dict):
            cleaned_data = {}
            for key, value in data.items():
                if isinstance(value, str):
                    cleaned_data[key] = normalize_text(value)
                else:
                    cleaned_data[key] = value
            return cleaned_data
        return data
    except Exception as e:
        print(f"Error fetching {endpoint} with {params}: {e}")
        return {}


def normalize_text(text):
    """Normalize text to handle encoding issues"""
    if not isinstance(text, str):
        return text

    # Replace escaped quotes and other common encoding issues
    replacements = {
        "\\'": "'",  # Replace escaped single quotes
        '\\"': '"',  # Replace escaped double quotes
        "\\\\": "\\",  # Replace double backslashes
        "\\/": "/",  # Replace escaped forward slashes
    }

    normalized = text
    for old, new in replacements.items():
        normalized = normalized.replace(old, new)

    # Additional normalization for common Indonesian characters
    normalized = normalized.replace("\\u0027", "'")  # Unicode single quote
    normalized = normalized.replace("\\u0022", '"')  # Unicode double quote

    return normalized


def clean_data_structure(data):
    """Recursively clean all text fields in the data structure"""
    if isinstance(data, dict):
        cleaned = {}
        for key, value in data.items():
            if key == "nama" and isinstance(value, str):
                cleaned[key] = normalize_text(value)
            else:
                cleaned[key] = clean_data_structure(value)
        return cleaned
    elif isinstance(data, list):
        return [clean_data_structure(item) for item in data]
    elif isinstance(data, str):
        return normalize_text(data)
    else:
        return data


def save_to_file(data, filename):
    # Clean the data before saving
    cleaned_data = clean_data_structure(data)
    with open(filename, "w", encoding="utf-8", errors="replace") as f:
        json.dump(cleaned_data, f, ensure_ascii=False, indent=2, separators=(",", ": "))


def load_checkpoint(checkpoint_file):
    """Load checkpoint data if exists, otherwise return empty structure"""
    if os.path.exists(checkpoint_file):
        try:
            with open(checkpoint_file, "r", encoding="utf-8") as f:
                data = json.load(f)
                print(f"📂 Checkpoint ditemukan: {checkpoint_file}")
                print(f"   - Provinsi yang sudah diproses: {len(data.get('pro', []))}")

                # Clean the loaded data to fix any encoding issues
                cleaned_data = clean_data_structure(data)
                return cleaned_data
        except Exception as e:
            print(f"⚠️ Error loading checkpoint: {e}")
    return {"pro": []}


def save_checkpoint(data, checkpoint_file, progress_info=""):
    """Save checkpoint with progress information"""
    try:
        save_to_file(data, checkpoint_file)
        if progress_info:
            print(f"💾 Checkpoint disimpan: {progress_info}")
    except Exception as e:
        print(f"⚠️ Error saving checkpoint: {e}")


def get_processed_ids(data, level="pro"):
    """Get list of already processed IDs at specified level"""
    if level == "pro":
        return [p["id"] for p in data.get("pro", [])]
    return []


def find_last_processed_position(data):
    """Find the last processed position to resume from"""
    provinces = data.get("pro", [])
    if not provinces:
        return None, None, None, None

    last_pro = provinces[-1]
    last_pro_id = last_pro["id"]

    kabupaten_list = last_pro.get("kab", [])
    if not kabupaten_list:
        return last_pro_id, None, None, None

    last_kab = kabupaten_list[-1]
    last_kab_id = last_kab["id"]

    kecamatan_list = last_kab.get("kec", [])
    if not kecamatan_list:
        return last_pro_id, last_kab_id, None, None

    last_kec = kecamatan_list[-1]
    last_kec_id = last_kec["id"]

    return last_pro_id, last_kab_id, last_kec_id, len(last_kec.get("des", []))


def scrape_all():
    global scraper_state

    try:
        # Set running flag and reset shutdown event
        scraper_state["is_running"] = True
        scraper_state["shutdown_event"].clear()

        # Get current date in YYYYMMDD format
        date_str = datetime.now().strftime("%Y%m%d")
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        # File paths
        checkpoint_file = f"output/checkpoints/checkpoint_{date_str}.json"
        temp_file = f"output/temp_wilayah_{timestamp}.json"
        final_file = f"output/wilayah_final_{date_str}.json"

        # Store in global state for signal handler
        scraper_state["checkpoint_file"] = checkpoint_file
        scraper_state["temp_file"] = temp_file

        # Load checkpoint if exists
        all_data = load_checkpoint(checkpoint_file)
        scraper_state["current_data"] = all_data  # Store in global state

        processed_pro_ids = get_processed_ids(all_data, "pro")

        # Find resume position
        resume_pro_id, resume_kab_id, resume_kec_id, resume_des_count = (
            find_last_processed_position(all_data)
        )

        if resume_pro_id:
            print(f"🔄 Melanjutkan dari provinsi: {resume_pro_id}")
            if resume_kab_id:
                print(f"   - Kabupaten: {resume_kab_id}")
            if resume_kec_id:
                print(f"   - Kecamatan: {resume_kec_id}")
                print(f"   - Desa sudah diproses: {resume_des_count}")

        print("📌 Mengambil data provinsi...")
        print("💡 Tekan Ctrl+C untuk menghentikan dan menyimpan checkpoint")
        print(f"🧵 Menggunakan {MAX_WORKERS} thread untuk parallel processing")

        provinsi_dict = get_json("list_pro", {"thn": THN})
        provinsi_items = list(provinsi_dict.items())

        # Filter out already processed provinces if resuming
        if resume_pro_id:
            # Find starting position
            start_index = 0
            for i, (pro_id, _) in enumerate(provinsi_items):
                if pro_id == resume_pro_id:
                    start_index = i
                    break
            provinsi_items = provinsi_items[start_index:]
        else:
            # Remove completely processed provinces
            provinsi_items = [
                (pid, pname)
                for pid, pname in provinsi_items
                if pid not in processed_pro_ids
            ]

        # Process provinces sequentially but kabupaten in parallel
        for i_pro, (pro_id, pro_name) in enumerate(
            tqdm(provinsi_items, desc="Provinsi", unit="provinsi")
        ):
            if scraper_state["shutdown_event"].is_set():
                break

            # Update global state periodically
            with data_lock:
                scraper_state["current_data"] = all_data

            # Find existing province data or create new
            existing_prov = None
            with data_lock:
                for prov in all_data["pro"]:
                    if prov["id"] == pro_id:
                        existing_prov = prov
                        break

            if existing_prov:
                prov = existing_prov
            else:
                prov = {"id": pro_id, "nama": pro_name, "kab": []}

            print(
                f"  ▶️ [{i_pro+1}/{len(provinsi_items)}] Mengambil kabupaten di provinsi '{pro_name}'..."
            )
            kabupaten_dict = get_json("list_kab", {"thn": THN, "pro": pro_id})
            kabupaten_items = list(kabupaten_dict.items())

            # Filter kabupaten if resuming
            processed_kab_ids = [k["id"] for k in prov.get("kab", [])]
            if resume_pro_id == pro_id and resume_kab_id:
                # Find starting position for kabupaten
                start_index = 0
                for i, (kab_id, _) in enumerate(kabupaten_items):
                    if kab_id == resume_kab_id:
                        start_index = i
                        break
                kabupaten_items = kabupaten_items[start_index:]
            else:
                # Remove completely processed kabupaten
                kabupaten_items = [
                    (kid, kname)
                    for kid, kname in kabupaten_items
                    if kid not in processed_kab_ids
                ]

            if not kabupaten_items:
                continue

            # Process kabupaten in parallel batches
            batch_size = min(MAX_WORKERS, len(kabupaten_items))

            with ThreadPoolExecutor(max_workers=batch_size) as executor:
                futures = []

                for j, kab_data in enumerate(kabupaten_items):
                    if scraper_state["shutdown_event"].is_set():
                        break

                    # Handle resume for first kabupaten
                    resume_info = None
                    if (
                        j == 0
                        and resume_pro_id == pro_id
                        and resume_kab_id == kab_data[0]
                    ):
                        resume_info = (resume_kec_id, resume_des_count)

                    future = executor.submit(
                        process_kabupaten_parallel,
                        kab_data,
                        (pro_id, pro_name),
                        resume_info,
                    )
                    futures.append((future, kab_data))

                # Collect results and update data
                completed_kab = []
                for future, kab_data in futures:
                    if scraper_state["shutdown_event"].is_set():
                        break

                    try:
                        result = future.result(
                            timeout=300
                        )  # 5 minute timeout per kabupaten
                        if result:
                            completed_kab.append(result)

                            # Save checkpoint after each kabupaten completion
                            temp_prov = {
                                "id": pro_id,
                                "nama": pro_name,
                                "kab": completed_kab,
                            }
                            update_global_data_safely(all_data, temp_prov)

                            safe_checkpoint_save(
                                all_data,
                                checkpoint_file,
                                f"Kabupaten {result['nama']} selesai",
                            )
                            save_to_file(all_data, temp_file)

                    except Exception as e:
                        print(f"❌ Error processing kabupaten {kab_data[1]}: {e}")

            # Final province update
            if completed_kab:
                final_prov = {"id": pro_id, "nama": pro_name, "kab": completed_kab}
                update_global_data_safely(all_data, final_prov)

                safe_checkpoint_save(
                    all_data, checkpoint_file, f"Provinsi {pro_name} selesai"
                )
                save_to_file(all_data, temp_file)

        # Save final result
        if not scraper_state["shutdown_event"].is_set():
            save_to_file(all_data, final_file)
            print(f"✅ Selesai!")
            print(f"   📁 File checkpoint: {checkpoint_file}")
            print(f"   📁 File temp: {temp_file}")
            print(f"   📁 File final: {final_file}")

            # Optionally remove checkpoint after successful completion
            try:
                if os.path.exists(checkpoint_file):
                    os.remove(checkpoint_file)
                    print(f"   🗑️ Checkpoint dihapus karena scraping selesai")
            except Exception as e:
                print(f"   ⚠️ Tidak bisa menghapus checkpoint: {e}")

    except KeyboardInterrupt:
        # This should be handled by signal handler, but just in case
        print("\n🛑 Scraping dihentikan oleh pengguna")

    finally:
        # Reset running state
        scraper_state["is_running"] = False
        scraper_state["shutdown_event"].set()
        scraper_state["current_data"] = None
        scraper_state["checkpoint_file"] = None
        scraper_state["temp_file"] = None


def show_checkpoint_info():
    """Show information about existing checkpoints"""
    checkpoint_dir = "output/checkpoints"
    if not os.path.exists(checkpoint_dir):
        print("📁 Tidak ada folder checkpoint")
        return

    checkpoints = [f for f in os.listdir(checkpoint_dir) if f.endswith(".json")]
    if not checkpoints:
        print("📁 Tidak ada checkpoint yang ditemukan")
        return

    print("📁 Checkpoint yang tersedia:")
    for cp in sorted(checkpoints):
        checkpoint_path = os.path.join(checkpoint_dir, cp)
        try:
            with open(checkpoint_path, "r", encoding="utf-8") as f:
                data = json.load(f)
                provinces = data.get("pro", [])
                total_kab = sum(len(p.get("kab", [])) for p in provinces)
                total_kec = sum(
                    len(k.get("kec", [])) for p in provinces for k in p.get("kab", [])
                )
                total_des = sum(
                    len(kc.get("des", []))
                    for p in provinces
                    for k in p.get("kab", [])
                    for kc in k.get("kec", [])
                )

                file_size = os.path.getsize(checkpoint_path) / 1024 / 1024  # MB
                mod_time = datetime.fromtimestamp(os.path.getmtime(checkpoint_path))

                print(f"   🗂️ {cp}")
                print(f"      - Provinsi: {len(provinces)}")
                print(f"      - Kabupaten: {total_kab}")
                print(f"      - Kecamatan: {total_kec}")
                print(f"      - Desa: {total_des}")
                print(f"      - Ukuran: {file_size:.2f} MB")
                print(
                    f"      - Terakhir diupdate: {mod_time.strftime('%Y-%m-%d %H:%M:%S')}"
                )
        except Exception as e:
            print(f"   ❌ {cp} - Error: {e}")


def clean_old_checkpoints(keep_days=7):
    """Clean checkpoints older than specified days"""
    checkpoint_dir = "output/checkpoints"
    if not os.path.exists(checkpoint_dir):
        return

    import time

    current_time = time.time()
    days_in_seconds = keep_days * 24 * 60 * 60

    cleaned = 0
    for filename in os.listdir(checkpoint_dir):
        if filename.endswith(".json"):
            file_path = os.path.join(checkpoint_dir, filename)
            file_age = current_time - os.path.getmtime(file_path)

            if file_age > days_in_seconds:
                try:
                    os.remove(file_path)
                    cleaned += 1
                    print(f"🗑️ Menghapus checkpoint lama: {filename}")
                except Exception as e:
                    print(f"⚠️ Gagal menghapus {filename}: {e}")

    if cleaned > 0:
        print(f"✅ {cleaned} checkpoint lama dihapus")
    else:
        print("ℹ️ Tidak ada checkpoint lama yang perlu dihapus")


def fix_existing_file(input_file, output_file=None):
    """Fix encoding issues in existing JSON files"""
    if not os.path.exists(input_file):
        print(f"❌ File tidak ditemukan: {input_file}")
        return False

    if output_file is None:
        # Create backup and overwrite original
        backup_file = input_file.replace(".json", "_backup.json")
        os.rename(input_file, backup_file)
        output_file = input_file
        print(f"📁 Backup dibuat: {backup_file}")

    try:
        print(f"🔧 Memperbaiki encoding: {input_file}")

        # Load data
        with open(
            backup_file if output_file == input_file else input_file,
            "r",
            encoding="utf-8",
        ) as f:
            data = json.load(f)

        # Clean data
        cleaned_data = clean_data_structure(data)

        # Save cleaned data
        save_to_file(cleaned_data, output_file)

        print(f"✅ File berhasil diperbaiki: {output_file}")
        return True

    except Exception as e:
        print(f"❌ Error memperbaiki file: {e}")
        if output_file == input_file and os.path.exists(backup_file):
            # Restore backup if failed
            os.rename(backup_file, input_file)
            print(f"🔄 Backup dikembalikan")
        return False


def safe_checkpoint_save(data, checkpoint_file, progress_info=""):
    """Thread-safe checkpoint saving"""
    with checkpoint_lock:
        try:
            save_to_file(data, checkpoint_file)
            if progress_info:
                with progress_lock:
                    print(f"💾 Checkpoint disimpan: {progress_info}")
        except Exception as e:
            with progress_lock:
                print(f"⚠️ Error saving checkpoint: {e}")


def process_kecamatan(kec_data, prov_info, kab_info, resume_info=None):
    """Process a single kecamatan in a thread"""
    if scraper_state["shutdown_event"].is_set():
        return None

    kec_id, kec_name = kec_data
    pro_id, pro_name = prov_info
    kab_id, kab_name = kab_info
    resume_des_count = resume_info if resume_info else 0

    try:
        with progress_lock:
            print(f"      🧵 Thread memproses kecamatan: {kec_name}")

        kec = {"id": kec_id, "nama": kec_name, "des": []}

        # Get desa data
        desa_dict = get_json(
            "list_des",
            {"thn": THN, "pro": pro_id, "kab": kab_id, "kec": kec_id},
        )

        if scraper_state["shutdown_event"].is_set():
            return None

        desa_items = list(desa_dict.items())

        # Process desa with resume capability
        start_des_index = resume_des_count if resume_des_count else 0

        for i_des, (des_id, des_name) in enumerate(desa_items[start_des_index:]):
            if scraper_state["shutdown_event"].is_set():
                break

            actual_index = start_des_index + i_des
            with progress_lock:
                print(
                    f"        🧵 [{actual_index+1}/{len(desa_items)}] Desa: {des_name}"
                )

            kec["des"].append({"id": des_id, "nama": des_name})

        return kec

    except Exception as e:
        with progress_lock:
            print(f"❌ Error processing kecamatan {kec_name}: {e}")
        return None


def process_kabupaten_parallel(kab_data, prov_info, resume_info=None):
    """Process kabupaten with parallel kecamatan processing"""
    if scraper_state["shutdown_event"].is_set():
        return None

    kab_id, kab_name = kab_data
    pro_id, pro_name = prov_info
    resume_kec_id, resume_des_count = resume_info if resume_info else (None, None)

    try:
        with progress_lock:
            print(f"    🧵 Thread memproses kabupaten: {kab_name}")

        kab = {"id": kab_id, "nama": kab_name, "kec": []}

        # Get kecamatan data
        kecamatan_dict = get_json(
            "list_kec", {"thn": THN, "pro": pro_id, "kab": kab_id}
        )

        if scraper_state["shutdown_event"].is_set():
            return None

        kecamatan_items = list(kecamatan_dict.items())

        # Filter kecamatan if resuming
        if resume_kec_id:
            start_index = 0
            for i, (kec_id, _) in enumerate(kecamatan_items):
                if kec_id == resume_kec_id:
                    start_index = i
                    break
            kecamatan_items = kecamatan_items[start_index:]

        # Process kecamatan in smaller batches to avoid overwhelming the API
        batch_size = min(2, len(kecamatan_items))  # Smaller batch for API courtesy

        for i in range(0, len(kecamatan_items), batch_size):
            if scraper_state["shutdown_event"].is_set():
                break

            batch = kecamatan_items[i : i + batch_size]

            with ThreadPoolExecutor(max_workers=min(batch_size, 2)) as executor:
                futures = []

                for j, kec_data in enumerate(batch):
                    if scraper_state["shutdown_event"].is_set():
                        break

                    # Handle resume for first kecamatan in batch
                    resume_des = None
                    if i == 0 and j == 0 and resume_kec_id == kec_data[0]:
                        resume_des = resume_des_count

                    future = executor.submit(
                        process_kecamatan,
                        kec_data,
                        prov_info,
                        (kab_id, kab_name),
                        resume_des,
                    )
                    futures.append(future)

                # Collect results
                for future in as_completed(futures):
                    if scraper_state["shutdown_event"].is_set():
                        break

                    result = future.result()
                    if result:
                        kab["kec"].append(result)

        return kab

    except Exception as e:
        with progress_lock:
            print(f"❌ Error processing kabupaten {kab_name}: {e}")
        return None


def update_global_data_safely(all_data, prov):
    """Thread-safe update of global data"""
    with data_lock:
        # Find existing province or add new one
        existing_prov = None
        for existing in all_data["pro"]:
            if existing["id"] == prov["id"]:
                existing_prov = existing
                break

        if existing_prov:
            # Merge kabupaten data
            existing_kab_ids = {k["id"] for k in existing_prov.get("kab", [])}
            for kab in prov["kab"]:
                if kab["id"] not in existing_kab_ids:
                    existing_prov["kab"].append(kab)
        else:
            all_data["pro"].append(prov)

        # Update global state
        scraper_state["current_data"] = all_data


def show_help():
    """Show help information"""
    print("🔧 Scraper API Wilayah Indonesia")
    print("=" * 50)
    print()
    print("📋 PERINTAH YANG TERSEDIA:")
    print()
    print("🚀 SCRAPING:")
    print("   python scrape_api_wilayah.py                    - Mulai scraping (default)")
    print("   python scrape_api_wilayah.py scrape             - Mulai/lanjutkan scraping")
    print("   python scrape_api_wilayah.py scrape [threads]   - Scraping dengan N threads (1-8)")
    print()
    print("📊 MANAGEMENT:")
    print("   python scrape_api_wilayah.py info               - Lihat info checkpoint")
    print("   python scrape_api_wilayah.py clean [days]       - Hapus checkpoint lama")
    print("   python scrape_api_wilayah.py fix <input> [out]  - Perbaiki encoding JSON")
    print()
    print("ℹ️ BANTUAN:")
    print("   python scrape_api_wilayah.py help               - Tampilkan bantuan ini")
    print("   python scrape_api_wilayah.py --help             - Tampilkan bantuan ini")
    print("   python scrape_api_wilayah.py -h                 - Tampilkan bantuan ini")
    print()
    print("📖 CONTOH PENGGUNAAN:")
    print("   python scrape_api_wilayah.py scrape 2           - Gunakan 2 threads")
    print("   python scrape_api_wilayah.py clean 3            - Hapus checkpoint >3 hari")
    print("   python scrape_api_wilayah.py fix data.json      - Perbaiki file data.json")
    print()
    print("🛑 STOP AMAN:")
    print("   Ctrl+C                                           - Hentikan dengan checkpoint")
    print()
    print("📁 FILE OUTPUT:")
    print("   output/checkpoints/checkpoint_YYYYMMDD.json     - Checkpoint harian")
    print("   output/temp_wilayah_YYYYMMDD_HHMMSS.json        - File temporary")
    print("   output/wilayah_final_YYYYMMDD.json              - Hasil akhir")
    print()
    print("📚 Dokumentasi lengkap: DOKUMENTASI_SCRAPER.md")
    print("⚡ Quick reference: QUICK_REFERENCE.md")


def set_max_workers(count):
    """Set the maximum number of worker threads"""
    global MAX_WORKERS
    MAX_WORKERS = count


if __name__ == "__main__":
    import sys

    if len(sys.argv) > 1:
        command = sys.argv[1].lower()

        if command == "info":
            show_checkpoint_info()
        elif command == "clean":
            days = 7
            if len(sys.argv) > 2:
                try:
                    days = int(sys.argv[2])
                except ValueError:
                    print("⚠️ Jumlah hari harus berupa angka")
                    sys.exit(1)
            clean_old_checkpoints(days)
        elif command == "scrape":
            # Check for thread count parameter
            if len(sys.argv) > 2:
                try:
                    thread_count = int(sys.argv[2])
                    if thread_count < 1 or thread_count > 8:
                        print("⚠️ Jumlah thread harus antara 1-8")
                        sys.exit(1)
                    set_max_workers(thread_count)
                    print(f"🧵 Menggunakan {thread_count} thread")
                except ValueError:
                    print("⚠️ Jumlah thread harus berupa angka")
                    sys.exit(1)
            scrape_all()
        elif command == "fix":
            if len(sys.argv) < 3:
                print(
                    "❌ Format: python scrape_api_wilayah.py fix <input_file> [output_file]"
                )
                sys.exit(1)

            input_file = sys.argv[2]
            output_file = sys.argv[3] if len(sys.argv) > 3 else None
            fix_existing_file(input_file, output_file)
        elif command == "help" or command == "--help" or command == "-h":
            show_help()
        else:
            print("❌ Perintah tidak dikenal. Gunakan 'help' untuk melihat perintah yang tersedia.")
            show_help()
    else:
        scrape_all()
