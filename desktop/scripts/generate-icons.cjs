// Build-only renderer. The checked-in SVG is the single design input.
const fs=require('node:fs');
const path=require('node:path');
const {Resvg}=require(path.resolve('out/icon-tools/node_modules/@resvg/resvg-js'));
const source=fs.readFileSync('frontend/icons/routilet.svg','utf8');
const out='out/icons';fs.mkdirSync(out,{recursive:true});
const cache=new Map();
function png(size){
 if(!cache.has(size))cache.set(size,Buffer.from(new Resvg(source,{fitTo:{mode:'width',value:size}}).render().asPng()));
 return cache.get(size);
}
fs.writeFileSync('frontend/icons/routilet.png',png(256));
const sizes=[16,24,32,48,64,128,256];
const header=Buffer.alloc(6+16*sizes.length);header.writeUInt16LE(1,2);header.writeUInt16LE(sizes.length,4);
let offset=header.length;const images=[];
for(let i=0;i<sizes.length;i++){
 const size=sizes[i],data=png(size),pos=6+i*16;
 header[pos]=header[pos+1]=size===256?0:size;
 header.writeUInt16LE(1,pos+4);header.writeUInt16LE(32,pos+6);
 header.writeUInt32LE(data.length,pos+8);header.writeUInt32LE(offset,pos+12);
 images.push(data);offset+=data.length;
}
fs.writeFileSync(path.join(out,'Routilet.ico'),Buffer.concat([header,...images]));
const iconset=path.join(out,'Routilet.iconset');fs.mkdirSync(iconset,{recursive:true});
for(const size of [16,32,128,256,512]){
 fs.writeFileSync(path.join(iconset,'icon_'+size+'x'+size+'.png'),png(size));
 fs.writeFileSync(path.join(iconset,'icon_'+size+'x'+size+'@2x.png'),png(size*2));
}
console.log('Rendered PNG, 7-size ICO and macOS iconset from SVG');
