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
  icon='<path d="M18 23l12-6 12 6v20l-12 6-12-6z M18 23l12 7 12-7 M30 30v19" fill="none" stroke="url(#gold)" stroke-width="1.5"/>'
  p=[gradient,f'<rect x="1" y="8" width="58" height="58" rx="14" fill="{bg}" stroke="{edge}"/>',icon,text(80,38,title,size,'url(#gold)',700),text(82,61,sec['caption'],10,sub),f'<path d="M1 82H999" stroke="{edge}"/>']
  (OUT/f'{sec["id"]}-{theme}.svg').write_text(svg(''.join(p),88),encoding='utf-8')
print('Generated gold visuals for '+CFG['repository'])
