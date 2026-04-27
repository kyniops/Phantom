import os
import ctypes
from ctypes import wintypes
import struct

# Windows API Constants
RT_ICON = 3
RT_GROUP_ICON = 14
RT_VERSION = 16
RT_MANIFEST = 24

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
    h_src = ctypes.windll.kernel32.LoadLibraryExW(src_exe, None, LOAD_LIBRARY_AS_DATAFILE)
    if not h_src:
        return False

    h_dst = ctypes.windll.kernel32.BeginUpdateResourceW(dst_exe, False)
    if not h_dst:
        ctypes.windll.kernel32.FreeLibrary(h_src)
        return False

    resource_types = [RT_ICON, RT_GROUP_ICON, RT_VERSION, RT_MANIFEST]
    
    # Callback function for EnumResourceNamesW
    ENUMRESNAMEPROC = ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HANDLE, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p)

    def enum_callback(hModule, lpType, lpName, lParam):
        # lpType and lpName can be integer IDs or pointers to strings
        
        # Load the resource
        hResInfo = ctypes.windll.kernel32.FindResourceW(hModule, lpName, lpType)
        if hResInfo:
            hResData = ctypes.windll.kernel32.LoadResource(hModule, hResInfo)
            if hResData:
                pResData = ctypes.windll.kernel32.LockResource(hResData)
                sizeRes = ctypes.windll.kernel32.SizeofResource(hModule, hResInfo)
                
                # Copy to destination (using 0x0409 for English)
                ctypes.windll.kernel32.UpdateResourceW(
                    h_dst, lpType, lpName, 0x0409,
                    pResData, sizeRes
                )
        return True

    callback = ENUMRESNAMEPROC(enum_callback)

    for res_type in resource_types:
        ctypes.windll.kernel32.EnumResourceNamesW(h_src, res_type, callback, 0)

    # Clean up
    ctypes.windll.kernel32.EndUpdateResourceW(h_dst, False)
    ctypes.windll.kernel32.FreeLibrary(h_src)
    return True

def inject_icon(exe_path, icon_path):
    """
    Inject a custom icon into a Windows executable using Windows API.
    """
    if not os.path.exists(exe_path):
        return False
    
    if not os.path.exists(icon_path):
        return False

    if os.name != 'nt':
        return False

    try:
        # Load the icon file
        with open(icon_path, 'rb') as f:
            data = f.read()

        # ICO Header
        reserved, ico_type, count = struct.unpack('<HHH', data[:6])
        if reserved != 0 or ico_type != 1:
            return False

        entries = []
        offset = 6
        for i in range(count):
            # ICONDIRENTRY
            width, height, colors, res, planes, bpp, size, image_offset = struct.unpack('<BBBBHHII', data[offset:offset+16])
            entries.append({
                'width': width, 'height': height, 'colors': colors, 'res': res,
                'planes': planes, 'bpp': bpp, 'size': size, 'offset': image_offset
            })
            offset += 16

        # Begin Update
        handle = ctypes.windll.kernel32.BeginUpdateResourceW(exe_path, False)
        if not handle:
            return False

        # 1. Write each ICON image
        for i, entry in enumerate(entries):
            icon_data = data[entry['offset'] : entry['offset'] + entry['size']]
            # Use i+1 as the icon ID
            res = ctypes.windll.kernel32.UpdateResourceW(
                handle, RT_ICON, i + 1, 0,
                icon_data, len(icon_data)
            )
            if not res:
                ctypes.windll.kernel32.EndUpdateResourceW(handle, True) # Cancel
                return False

        # 2. Write Group Icon
        grp_header = GRPICONDIR().pack(0, 1, count)
        grp_data = bytearray(grp_header)
        
        for i, entry in enumerate(entries):
            # GRPICONDIRENTRY is slightly different from ICONDIRENTRY (replaces offset with ID)
            entry_data = struct.pack('<BBBBHHII', 
                entry['width'], entry['height'], entry['colors'], entry['res'],
                entry['planes'], entry['bpp'], entry['size'], i + 1
            )
            grp_data.extend(entry_data)

        # Use 1 as the Group Icon ID (standard for main icon)
        res = ctypes.windll.kernel32.UpdateResourceW(
            handle, RT_GROUP_ICON, 1, 0,
            bytes(grp_data), len(grp_data)
        )
        
        if not res:
            ctypes.windll.kernel32.EndUpdateResourceW(handle, True) # Cancel
            return False

        # End Update
        if not ctypes.windll.kernel32.EndUpdateResourceW(handle, False):
            return False

        return True
    except Exception as e:
        print(f"Error injecting icon: {e}")
        return False
