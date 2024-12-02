import svgwrite
import os

w = 40
h = 16
stroke_width = 0

def curve_sleepers(dwg, g):
    steps = 7

    rot_center = (200 - h - stroke_width, 200)
    rot_center = (200, 200)
    sleepers = g.add(dwg.g(id='sleepers', fill='none'))
    offset = h + stroke_width/2
    for i in range(steps):
        s = dwg.rect(
            insert=((200 - w) / 2, 200 - h/2),
            size=(w, h),
            fill="#8B4513",
            stroke="black",
            stroke_width=stroke_width
        )
        s.rotate(90.0 / (steps - 1) * i, center=rot_center)
        sleepers.add(s)

def curve_rails(dwg, g):
    g.add(dwg.path(d="M90,200 A 110 110 0 0 1 200,90", fill="none", stroke="black", stroke_width=6))
    g.add(dwg.path(d="M110,200 A 90 90 0 0 1 200,110", fill="none", stroke="black", stroke_width=6))

def curve(dwg, g):
    curve_sleepers(dwg, g)
    curve_rails(dwg, g)

def straight_sleepers(dwg, g):
    steps = 10

    sleepers = g.add(dwg.g(id='sleepers2', fill='none'))
    x = (200 - w) / 2
    for i in range(steps):
        s = dwg.rect(
            insert=(x, 200 - h/2),
            size=(w, h),
            fill="#8B4513",
            stroke="black",
            stroke_width=stroke_width
        )
        s.translate(0, -200 / (steps - 1) * i)
        sleepers.add(s)

def straight_rails(dwg, g):
    g.add(dwg.line(start=(90,200), end=(90, 0), fill="none", stroke="black", stroke_width=6))
    g.add(dwg.line(start=(110,200), end=(110, 0), fill="none", stroke="black", stroke_width=6))

def straight(dwg, g):
    straight_sleepers(dwg, g)
    straight_rails(dwg, g)

def init(name):
    dwg = svgwrite.Drawing(f"tiles/{name}.svg", size=(200,200), profile='full')
    dwg.add(dwg.rect(size=(200, 200), fill="lightgreen"))
    g = dwg.g()
    dwg.add(g)
    return (dwg, g)

def save(dwg):
    print(f">>>>> {dwg.filename}")
    dwg.save()
    fn = dwg.filename
    base, _ = os.path.splitext(fn)
    os.system(f"sips -s format png {fn} --out {base}.png")
    os.system(f"rm {fn}")

dwg, g = init("empty")
save(dwg)

for rot in [0, 90, 180, 270]:
    dwg, g = init(f"curve-{rot}")
    curve(dwg, g)
    g.rotate(rot, center=(100, 100))
    save(dwg)

for rot in [0, 90]:
    dwg, g = init(f"straight-{rot}")
    straight(dwg, g)
    g.rotate(rot, center=(100, 100))
    save(dwg)

dwg, g = init(f"cross")
straight_sleepers(dwg, g)
g2 = dwg.g()
dwg.add(g2)
straight(dwg, g2)
g2.rotate(90, center=(100, 100))
straight_rails(dwg, dwg)
save(dwg)

for i in range(4):
    dwg, g = init(f"turnout-{i}")
    curve_sleepers(dwg, g)
    straight(dwg, g)
    curve_rails(dwg, g)
    g.rotate(270, center=(100, 100))
    if i % 2 == 1:
        g.scale(1, -1)
        g.translate(0, -200)
    if i > 1:
        g.rotate(90, center=(100, 100))
    save(dwg)

#
