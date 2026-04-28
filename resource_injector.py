import os
import ctypes
from ctypes import wintypes
import struct

# Windows API Constants
RT_ICON = 3
RT_GROUP_ICON = 14
RT_VERSION = 16
RT_MANIFEST = 24

class GRPICONDIR:
    def pack(self, reserved, type, count):
        return struct.pack('<HHH', reserved, type, count)

# Windows API Setup
kernel32 = ctypes.windll.kernel32

# Set argument types for better stability
kernel32.BeginUpdateResourceW.argtypes = [wintypes.LPCWSTR, wintypes.BOOL]
kernel32.BeginUpdateResourceW.restype = wintypes.HANDLE

kernel32.UpdateResourceW.argtypes = [wintypes.HANDLE, wintypes.LPCWSTR, wintypes.LPCWSTR, wintypes.WORD, wintypes.LPVOID, wintypes.DWORD]
kernel32.UpdateResourceW.restype = wintypes.BOOL

kernel32.EndUpdateResourceW.argtypes = [wintypes.HANDLE, wintypes.BOOL]
kernel32.EndUpdateResourceW.restype = wintypes.BOOL

# Helper to handle resource IDs (MAKEINTRESOURCE equivalent)
def MAKEINTRESOURCEW(i):
    return ctypes.cast(wintypes.LPWSTR(i), wintypes.LPCWSTR)

# Resource cloning helpers
def clone_resources(src_exe, dst_exe):
    """
    Clones Icon, Version Info, and Manifest from src_exe to dst_exe.
    """
    if not os.path.exists(src_exe) or not os.path.exists(dst_exe):
        return False

    if os.name != 'nt':
        return False

    # Load source library as data
    LOAD_LIBRARY_AS_DATAFILE = 0x00000002
    h_src = kernel32.LoadLibraryExW(src_exe, None, LOAD_LIBRARY_AS_DATAFILE)
    if not h_src:
        return False

    h_dst = kernel32.BeginUpdateResourceW(dst_exe, False)
    if not h_dst:
        kernel32.FreeLibrary(h_src)
        return False

    resource_types = [RT_ICON, RT_GROUP_ICON, RT_VERSION, RT_MANIFEST]
    
    # Callback function for EnumResourceNamesW
    ENUMRESNAMEPROC = ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HANDLE, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p)

    def enum_callback(hModule, lpType, lpName, lParam):
        # Load the resource
        hResInfo = kernel32.FindResourceW(hModule, lpName, lpType)
        if hResInfo:
            hResData = kernel32.LoadResource(hModule, hResInfo)
            if hResData:
                pResData = kernel32.LockResource(hResData)
                sizeRes = kernel32.SizeofResource(hModule, hResInfo)
                
                # Copy to destination (using 0x0409 for English)
                kernel32.UpdateResourceW(
                    h_dst, lpType, lpName, 0x0409,
                    pResData, sizeRes
                )
        return True

    callback = ENUMRESNAMEPROC(enum_callback)

    for res_type in resource_types:
        type_ptr = MAKEINTRESOURCEW(res_type)
        kernel32.EnumResourceNamesW(h_src, type_ptr, callback, 0)

    # Clean up
    kernel32.EndUpdateResourceW(h_dst, False)
    kernel32.FreeLibrary(h_src)
    return True

def inject_icon(exe_path, icon_path):
    """
    Inject a custom icon into a Windows executable using Windows API.
    """
    import time
    
    if not os.path.exists(exe_path):
        return False
    
    if not os.path.exists(icon_path):
        return False

    if os.name != 'nt':
        return False

    # Retry mechanism in case the file is locked by antivirus
    handle = None
    for attempt in range(5):
        handle = kernel32.BeginUpdateResourceW(exe_path, False)
        if handle:
            break
        time.sleep(0.5)

    if not handle:
        print(f"Error: BeginUpdateResourceW failed (Last error: {ctypes.get_last_error()})")
        return False

    try:
        # Load the icon file
        with open(icon_path, 'rb') as f:
            data = f.read()

        if len(data) < 6:
            kernel32.EndUpdateResourceW(handle, True)
            return False

        reserved, ico_type, count = struct.unpack('<HHH', data[:6])
        if reserved != 0 or ico_type != 1:
            kernel32.EndUpdateResourceW(handle, True)
            return False

        entries = []
        offset = 6
        for i in range(count):
            if len(data) < offset + 16: break
            width, height, colors, res, planes, bpp, size, image_offset = struct.unpack('<BBBBHHII', data[offset:offset+16])
            entries.append({
                'width': width, 'height': height, 'colors': colors, 'res': res,
                'planes': planes, 'bpp': bpp, 'size': size, 'offset': image_offset
            })
            offset += 16

        # 1. Write each ICON image
        for i, entry in enumerate(entries):
            icon_data = data[entry['offset'] : entry['offset'] + entry['size']]
            res = kernel32.UpdateResourceW(
                handle, MAKEINTRESOURCEW(RT_ICON), MAKEINTRESOURCEW(i + 1), 0,
                icon_data, len(icon_data)
            )
            if not res:
                kernel32.EndUpdateResourceW(handle, True)
                return False

        # 2. Write Group Icon
        grp_header = GRPICONDIR().pack(0, 1, count)
        grp_data = bytearray(grp_header)
        for i, entry in enumerate(entries):
            entry_data = struct.pack('<BBBBHHIH', 
                entry['width'], entry['height'], entry['colors'], entry['res'],
                entry['planes'], entry['bpp'], entry['size'], i + 1
            )
            grp_data.extend(entry_data)

        res = kernel32.UpdateResourceW(
            handle, MAKEINTRESOURCEW(RT_GROUP_ICON), MAKEINTRESOURCEW(1), 0,
            bytes(grp_data), len(grp_data)
        )
        
        if not res:
            kernel32.EndUpdateResourceW(handle, True)
            return False

        # End Update
        if not kernel32.EndUpdateResourceW(handle, False):
            return False

        return True
    except Exception as e:
        if handle: kernel32.EndUpdateResourceW(handle, True)
        print(f"Exception in inject_icon: {e}")
        return False
