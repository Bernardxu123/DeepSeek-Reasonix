/**
 * Frame Protocol TypeScript Types
 * 
 * Wire format for React ↔ Rust rendering bridge.
 * Must match the Rust frame_protocol module exactly.
 */

export interface FrameProtocol {
  v: number; // version
  seq: number; // sequence number for deduplication
  w: number; // width in cells
  h: number; // height in cells
  root: FrameNode;
  dirty?: DirtyRect[]; // optional dirty rectangles for incremental rendering
}

export interface DirtyRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export type FrameNode = TextNode | BoxNode;

interface TextNode {
  t: 'text';
  runs: TextRun[];
  wrap?: WrapMode;
}

interface BoxNode {
  t: 'box';
  layout?: BoxLayout;
  children: FrameNode[];
}

export interface TextRun {
  txt: string;
  style?: TextStyle;
}

export interface TextStyle {
  fg?: Color;
  bg?: Color;
  bold?: boolean;
  dim?: boolean;
  italic?: boolean;
  underline?: boolean;
  inverse?: boolean;
  strikethrough?: boolean;
}

export type Color = NamedColor | HexColor | Ansi256Color;

interface NamedColor {
  named: string;
}

interface HexColor {
  hex: string;
}

interface Ansi256Color {
  ansi256: number;
}

export type WrapMode = 'wrap' | 'truncate' | 'truncateStart' | 'truncateMiddle' | 'none';

export interface BoxLayout {
  direction?: FlexDirection;
  gap?: number;
  px?: number; // padding-x
  py?: number; // padding-y
  mx?: number; // margin-x
  my?: number; // margin-y
  width?: Dimension;
  height?: Dimension;
  flexGrow?: number;
  flexShrink?: number;
}

export type FlexDirection = 'row' | 'column';

export type Dimension = CellsDimension | FillDimension;

interface CellsDimension {
  cells: number;
}

interface FillDimension {
  fill: true;
}

/** Schema version - must match Rust SCHEMA_VERSION constant */
export const FRAME_PROTOCOL_VERSION = 1;
