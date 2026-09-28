import argparse
import os
from pathlib import Path

from fpdf import FPDF


PLACEHOLDERS = [
    "BOT_URL",
    "REPOSITORY_URL",
    "COMMIT_HASH",
    "TEST_OWNER_ID",
    "TEST_ADMIN_ID",
    "TEST_MEMBER_ID",
    "MAX_BOT_TOKEN",
    "WEBHOOK_SECRET",
    "TEAM",
    "PILOT_TEAM",
]


class Presentation(FPDF):
    def footer(self):
        self.set_y(-15)
        self.set_font("Arial", "", 9)
        self.set_text_color(120, 120, 120)
        self.cell(0, 10, f"TaskFlow Kanban — {self.page_no()}/{{nb}}", align="C")


def load_slides(path):
    title = "TaskFlow Kanban"
    slides = []
    current = None
    in_code = False
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.rstrip()
        if line.startswith("# ") and current is None:
            title = line[2:].strip()
            continue
        if line.startswith("## "):
            current = {"title": line[3:].strip(), "blocks": []}
            slides.append(current)
            continue
        if current is None:
            continue
        if line.strip().startswith("```"):
            in_code = not in_code
            current["blocks"].append(("code-boundary", ""))
            continue
        if in_code:
            current["blocks"].append(("code", line))
        elif line.lstrip().startswith("- "):
            text = line.lstrip()[2:].strip()
            if line.startswith("  "):
                text = "    - " + text
            current["blocks"].append(("bullet", text))
        elif line.strip() == "":
            current["blocks"].append(("gap", ""))
        else:
            current["blocks"].append(("text", line.strip()))
    return title, slides


def substitute(text):
    for name in PLACEHOLDERS:
        text = text.replace(f"[{name}]", os.environ.get(name, f"[{name}]"))
    return text


def render_slide(pdf, title, blocks):
    pdf.add_page()
    pdf.set_font("Arial", "B", 24)
    pdf.set_text_color(20, 20, 20)
    pdf.multi_cell(pdf.epw, 11, substitute(title))
    pdf.ln(3)
    pdf.set_font("Arial", "", 15)
    pdf.set_text_color(35, 35, 35)
    for kind, text in blocks:
        if kind == "gap":
            pdf.ln(2)
        elif kind == "bullet":
            pdf.multi_cell(pdf.epw, 8, chr(8226) + "  " + substitute(text), markdown=True)
        elif kind == "code":
            pdf.set_font("Courier", "", 11)
            pdf.multi_cell(pdf.epw, 6, text)
            pdf.set_font("Arial", "", 15)
        elif kind == "text" and text:
            pdf.multi_cell(pdf.epw, 8, substitute(text), markdown=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", default=str(Path(__file__).with_name("presentation.md")))
    parser.add_argument("--output", default=str(Path(__file__).with_name("presentation.pdf")))
    args = parser.parse_args()
    source = Path(args.source)
    title, slides = load_slides(source)
    pdf = Presentation(orientation="L", format="A4")
    pdf.alias_nb_pages("{nb}")
    pdf.set_auto_page_break(True, margin=18)
    pdf.add_font("Arial", "", r"C:\Windows\Fonts\arial.ttf")
    pdf.add_font("Arial", "B", r"C:\Windows\Fonts\arialbd.ttf")
    pdf.add_page()
    pdf.set_font("Arial", "B", 34)
    pdf.multi_cell(pdf.epw, 15, title)
    pdf.set_font("Arial", "", 16)
    pdf.multi_cell(pdf.epw, 9, "Чат-бот MAX и мини-приложение для рабочих задач")
    for slide in slides:
        render_slide(pdf, slide["title"], slide["blocks"])
    pdf.output(args.output)


if __name__ == "__main__":
    main()
