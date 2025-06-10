from io import BytesIO

from PIL import Image


class _ImageConverter:
    def to_webp(self, input: BytesIO) -> bytes:
        with Image.open(input) as img:
            if img.mode in ("P", "L"):
                img = img.convert("RGB")
            elif img.mode == "PA":
                img = img.convert("RGBA")
            output_buffer = BytesIO()
            img.save(
                output_buffer,
                format="WEBP",
                quality=1,
                method=6
            )
            return output_buffer.getvalue()


converter = _ImageConverter()
