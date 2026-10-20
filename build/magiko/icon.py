"""Generate Magiko's monogram icon; requires Pillow only when regenerating."""
from pathlib import Path
from PIL import Image, ImageDraw
root = Path(__file__).resolve().parent
s = 1024
im = Image.new('RGBA', (s, s))
d = ImageDraw.Draw(im)
d.rounded_rectangle((48, 48, 976, 976), 210, fill='#19152f')
d.rounded_rectangle((66, 66, 958, 958), 195, outline='#7250bd', width=4)
d.line([(250, 725), (250, 345), (510, 600), (770, 345), (770, 725)], fill='#d9c6ff', width=76, joint='curve')
d.polygon([(735, 163), (756, 230), (823, 251), (756, 272), (735, 339), (714, 272), (647, 251), (714, 230)], fill='#72f0cd')
im.save(root/'icon-1024.png')
for size in (64, 256):
    im.resize((size,size), Image.Resampling.LANCZOS).save(root/f'icon-{size}.png')
im.save(root/'magiko.icns', format='ICNS')
