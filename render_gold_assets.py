"""Regenerate the repository's gold README visuals. Python standard library only."""
from pathlib import Path
import json,html
ROOT=Path(__file__).resolve().parent
CFG=json.loads((ROOT/'gold-theme.json').read_text(encoding='utf-8'))
OUT=ROOT/'docs/showcase';OUT.mkdir(parents=True,exist_ok=True)
def esc(s):return html.escape(str(s),quote=True)
def text(x,y,s,size,color,weight=400):
 return f'<text x="{x}" y="{y}" font-family="Segoe UI,Microsoft YaHei,Arial,sans-serif" font-size="{size}" font-weight="{weight}" fill="{color}">{esc(s)}</text>'
def svg(body,h):return f'<svg xmlns="http://www.w3.org/2000/svg" width="1000" height="{h}" viewBox="0 0 1000 {h}" role="img">'+body+'</svg>\n'
ICONS={'preview': 'M17 20h27v27H17z M17 27h27 M21 23h1 M25 23h1 M29 23h1 M22 33h7v8h-7z M33 33h6 M33 38h6', 'architecture': 'M24 17h12v10H24z M13 40h12v10H13z M35 40h12v10H35z M30 27v7 M19 40v-6h22v6', 'start': 'M22 18l24 16-24 16z', 'security': 'M30 17l15 6v12c0 8-8 14-15 17-7-3-15-9-15-17V23z M23 34l5 5 10-12', 'stack': 'M15 26l15-9 15 9-15 9z M15 34l15 9 15-9 M15 42l15 9 15-9', 'pipeline': 'M12 28h10v12H12z M25 28h10v12H25z M38 28h10v12H38z M22 34h3 M35 34h3', 'database': 'M16 23a14 6 0 1 0 28 0a14 6 0 1 0-28 0 M16 23v22a14 6 0 0 0 28 0V23 M16 34a14 6 0 0 0 28 0', 'config': 'M16 22h28 M16 34h28 M16 46h28 M23 18v8 M37 30v8 M27 42v8', 'payment': 'M14 22h32v25H14z M14 29h32 M20 40h7 M32 40h8', 'api': 'M22 24l-11 10 11 10 M38 24l11 10-11 10 M34 19l-8 30', 'deploy': 'M17 19h26v12H17z M17 38h26v12H17z M21 25h2 M21 44h2 M30 31v7', 'performance': 'M15 46V20 M15 46h32 M22 42V32 M31 42V25 M40 42V18', 'guide': 'M14 20h12l4 4 4-4h12v28H34l-4 3-4-3H14z M30 24v27', 'roadmap': 'M17 18v32 M17 22h25l-5 8 5 8H17', 'license': 'M21 17h21v32H21z M26 25h11 M26 31h11 M26 37h6 M16 23v27h20', 'collaboration': 'M15 26a5 5 0 1 0 10 0a5 5 0 1 0-10 0 M35 26a5 5 0 1 0 10 0a5 5 0 1 0-10 0 M11 47v-5a9 9 0 0 1 18 0v5 M31 47v-5a9 9 0 0 1 18 0v5', 'folder': 'M13 23h15l4 5h16v22H13z M13 23v-5h15l4 5', 'features': 'M20 18h20v31H20z M25 25l2 2 4-5 M25 35l2 2 4-5 M34 26h2 M34 36h2', 'question': 'M16 29a14 14 0 1 1 28 0c0 7-14 8-14 15 M30 49v1', 'operations': 'M19 19l7 7-6 6-7-7v8l8 8 7-1 13 13 6-6-13-13 1-7-8-8z', 'network': 'M30 17v16 M14 33h32 M14 33v9 M30 33v9 M46 33v9 M24 17h12v9H24z M9 42h10v9H9z M25 42h10v9H25z M41 42h10v9H41z'}
for theme in ['light','dark']:
 dark=theme=='dark'
 bg='#17140D' if dark else '#FAF8F3';edge='#51452B' if dark else '#DFD8C8';sub='#B8AC90' if dark else '#71674F'
 colors=['#C59938','#F8DF97','#D2AE50'] if dark else ['#765018','#BA8B24','#8B641A']
 gradient='<defs><linearGradient id="gold" x1="0%" y1="0%" x2="100%" y2="25%">'+''.join(f'<stop offset="{o}" stop-color="{c}"/>' for o,c in zip(['0%','48%','100%'],colors))+'</linearGradient><linearGradient id="shine"><stop stop-color="#FFF3C8" stop-opacity="0"/><stop offset=".48" stop-color="#D6B960" stop-opacity=".15"/><stop offset=".53" stop-color="#FFF8E4" stop-opacity=".7"/><stop offset=".59" stop-color="#D6B960" stop-opacity=".12"/><stop offset="1" stop-color="#FFF3C8" stop-opacity="0"/></linearGradient><clipPath id="hero"><rect x="1" y="1" width="998" height="258" rx="18"/></clipPath></defs>'
 motion='<style>@keyframes glint{from{transform:translateX(-280px)}to{transform:translateX(1260px)}}.glint{animation:glint 12s linear infinite}@media(prefers-reduced-motion:reduce){.glint{display:none}}</style>'
 title=CFG['title'];size=min(46, int(890/max(1,len(title)*.58)))
 p=[gradient,motion,f'<rect x=".5" y=".5" width="999" height="259" rx="18" fill="{bg}" stroke="{edge}"/>']
 p.append('<g clip-path="url(#hero)"><g class="glint"><rect x="0" y="-70" width="260" height="400" transform="skewX(-18)" fill="url(#shine)"/></g></g>')
 p.append(text(42,40,'ENGINEERING / OPEN SOURCE',12,sub,600))
 p.append(text(42,105,title,size,'url(#gold)',800))
 p.append(text(44,152,CFG['subtitle'],24,'url(#gold)',600))
 p.append(text(44,192,CFG['stack'],15,sub))
 p.append(f'<path d="M44 214H956" stroke="{edge}"/>')
 p.append(text(44,242,CFG['repository'],11,sub))
 (OUT/f'hero-{theme}.svg').write_text(svg(''.join(p),260),encoding='utf-8')
 for sec in CFG['sections']:
  title=sec['title'];size=min(27,int(820/max(1,sum(1 if ord(c)<128 else 1.8 for c in title)*.6)))
  icon='<path d="'+ICONS.get(sec.get('icon','guide'),ICONS['guide'])+'" fill="none" stroke="url(#gold)" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/>'
  p=[gradient,f'<rect x="1" y="8" width="58" height="58" rx="14" fill="{bg}" stroke="{edge}"/>',icon,text(80,38,title,size,'url(#gold)',700),text(82,61,sec['caption'],10,sub),f'<path d="M1 82H999" stroke="{edge}"/>']
  (OUT/f'{sec["id"]}-{theme}.svg').write_text(svg(''.join(p),88),encoding='utf-8')
print('Generated gold visuals for '+CFG['repository'])
