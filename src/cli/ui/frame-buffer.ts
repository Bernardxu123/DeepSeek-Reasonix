/**
 * Frame Buffer: Off-screen rendering with dirty-rect tracking
 * 
 * Solves flicker by:
 * 1. Batch updates at fixed 60fps cap
 * 2. Track dirty rectangles for incremental redraws
 * 3. Double-buffering to prevent tear
 */

import type { Cell, Frame } from '../frame/types.js';

export interface DirtyRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface FrameBufferConfig {
  width: number;
  height: number;
  fpsCap?: number; // Default 60
}

export class FrameBuffer {
  private front: Cell[][] = [];
  private back: Cell[][] = [];
  private dirtyRects: DirtyRect[] = [];
  private width: number;
  private height: number;
  private frameInterval: number;
  private lastFrameTime: number = 0;
  private sequence: number = 0;

  constructor(config: FrameBufferConfig) {
    this.width = config.width;
    this.height = config.height;
    this.frameInterval = 1000 / (config.fpsCap ?? 60);
    this.front = this.createBlankFrame();
    this.back = this.createBlankFrame();
  }

  private createBlankFrame(): Cell[][] {
    const blank: Cell[] = Array.from({ length: this.width }, () => ({
      char: ' ',
      width: 1,
    }));
    return Array.from({ length: this.height }, () => [...blank]);
  }

  resize(width: number, height: number) {
    this.width = width;
    this.height = height;
    this.front = this.createBlankFrame();
    this.back = this.createBlankFrame();
    this.dirtyRects = [{ x: 0, y: 0, w: width, h: height }];
  }

  /** Mark a region as dirty */
  markDirty(x: number, y: number, w: number, h: number) {
    const rect: DirtyRect = {
      x: Math.max(0, x),
      y: Math.max(0, y),
      w: Math.min(w, this.width - x),
      h: Math.min(h, this.height - y),
    };
    
    if (rect.w <= 0 || rect.h <= 0) return;
    
    // Try to merge with existing dirty rects
    let merged = false;
    for (let i = 0; i < this.dirtyRects.length; i++) {
      const existing = this.dirtyRects[i]!;
      if (this.rectsIntersect(existing, rect)) {
        this.dirtyRects[i] = this.mergeRects(existing, rect);
        merged = true;
        break;
      }
    }
    
    if (!merged) {
      this.dirtyRects.push(rect);
    }
  }

  private rectsIntersect(a: DirtyRect, b: DirtyRect): boolean {
    return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
  }

  private mergeRects(a: DirtyRect, b: DirtyRect): DirtyRect {
    const x = Math.min(a.x, b.x);
    const y = Math.min(a.y, b.y);
    const w = Math.max(a.x + a.w, b.x + b.w) - x;
    const h = Math.max(a.y + a.h, b.y + b.h) - y;
    return { x, y, w, h };
  }

  /** Write text to buffer at position */
  writeText(text: string, x: number, y: number, style?: Partial<Cell>) {
    if (y < 0 || y >= this.height) return;
    
    let cursor = x;
    for (const char of text) {
      if (cursor >= this.width) break;
      if (cursor < 0) {
        cursor++;
        continue;
      }
      
      const cell: Cell = {
        char,
        width: 1,
        ...style,
      };
      
      if (this.back[y]?.[cursor]) {
        const oldCell = this.back[y]![cursor]!;
        if (oldCell.char !== char || !this.cellsEqual(oldCell, cell)) {
          this.back[y]![cursor] = cell;
          this.markDirty(cursor, y, 1, 1);
        }
      }
      cursor++;
    }
  }

  private cellsEqual(a: Cell, b: Cell): boolean {
    return (
      a.fg === b.fg &&
      a.bg === b.bg &&
      a.bold === b.bold &&
      a.dim === b.dim &&
      a.italic === b.italic &&
      a.underline === b.underline &&
      a.inverse === b.inverse
    );
  }

  /** Fill entire row */
  fillRow(y: number, cells: Cell[]) {
    if (y < 0 || y >= this.height) return;
    this.back[y] = cells.slice(0, this.width);
    this.markDirty(0, y, this.width, 1);
  }

  /** Commit back buffer and get dirty regions */
  commit(now: number = Date.now()): { frame: Frame; dirty: DirtyRect[] } | null {
    // FPS cap
    if (now - this.lastFrameTime < this.frameInterval) {
      return null;
    }
    
    if (this.dirtyRects.length === 0) {
      return null;
    }

    // Swap buffers
    [this.front, this.back] = [this.back, this.front];
    
    const dirty = [...this.dirtyRects];
    this.dirtyRects = [];
    this.lastFrameTime = now;
    this.sequence++;

    const frame: Frame = {
      width: this.width,
      rows: this.front.map((row) => [...row]),
    };

    return { frame, dirty };
  }

  /** Force full redraw */
  invalidate() {
    this.dirtyRects = [{ x: 0, y: 0, w: this.width, h: this.height }];
  }

  /** Get current sequence number */
  getSequence(): number {
    return this.sequence;
  }

  /** Clear all dirty rects without committing */
  clearDirty() {
    this.dirtyRects = [];
  }
}

/** Global singleton for CLI app */
let globalBuffer: FrameBuffer | null = null;

export function getGlobalBuffer(): FrameBuffer | null {
  return globalBuffer;
}

export function initGlobalBuffer(config: FrameBufferConfig): FrameBuffer {
  globalBuffer = new FrameBuffer(config);
  return globalBuffer;
}
