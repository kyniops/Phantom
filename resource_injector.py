import os
import ctypes
from ctypes import wintypes
import struct

# Windows API Constants
RT_ICON = 3
RT_GROUP_ICON = 14

# Structures for ICO parsing and Resource updating
class GRPICONDIRENTRY(struct.Struct):
    def __init__(self):
        super().__init__('<BBBBHHII')
    def pack(self, width, height, colors, reserved, planes, bpp, size, id):
        return super().pack(width, height, colors, reserved, planes, bpp, size, id)

class GRPICONDIR(struct.Struct):
    def __init__(self):
        super().__init__('<HHH')
    def pack(self, reserved, type, count):
        return super().pack(reserved, type, count)

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
