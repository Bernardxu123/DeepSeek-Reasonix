/// Frame Protocol: JSON serialization layer for React ↔ Rust rendering bridge.
/// 
/// This module defines the wire format for sending scene descriptions from
/// TypeScript (React/Ink) to Rust (ratatui/crossterm) for high-performance rendering.
/// 
/// Design principles:
/// - Minimal allocation: reuse buffers where possible
/// - Incremental updates: support dirty-rect tracking
/// - Schema versioning: allow future evolution

use serde::{Deserialize, Serialize};

/// Schema version for backward compatibility
pub const SCHEMA_VERSION: u32 = 1;

/// A complete frame to be rendered. Sent as a single JSON line.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FrameProtocol {
    #[serde(rename = "v")]
    pub version: u32,
    
    #[serde(rename = "seq")]
    pub sequence: u64,
    
    /// Terminal dimensions in cells
    #[serde(rename = "w")]
    pub width: u16,
    
    #[serde(rename = "h")]
    pub height: u16,
    
    /// The scene graph to render
    pub root: FrameNode,
    
    /// Optional: dirty rectangles for incremental rendering
    /// If None, full screen redraw is required
    #[serde(rename = "dirty", default, skip_serializing_if = "Option::is_none")]
    pub dirty_rects: Option<Vec<DirtyRect>>,
}

/// A dirty rectangle for incremental updates
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct DirtyRect {
    #[serde(rename = "x")]
    pub x: u16,
    #[serde(rename = "y")]
    pub y: u16,
    #[serde(rename = "w")]
    pub width: u16,
    #[serde(rename = "h")]
    pub height: u16,
}

impl DirtyRect {
    pub fn new(x: u16, y: u16, width: u16, height: u16) -> Self {
        Self { x, y, width, height }
    }
    
    /// Check if this rect intersects with another
    pub fn intersects(&self, other: &DirtyRect) -> bool {
        self.x < other.x + other.width
            && self.x + self.width > other.x
            && self.y < other.y + other.height
            && self.y + self.height > other.y
    }
    
    /// Merge two intersecting rects into one bounding rect
    pub fn merge(&self, other: &DirtyRect) -> Option<DirtyRect> {
        if !self.intersects(other) {
            return None;
        }
        let x = self.x.min(other.x);
        let y = self.y.min(other.y);
        let w = (self.x + self.width).max(other.x + other.width) - x;
        let h = (self.y + self.height).max(other.y + other.height) - y;
        Some(DirtyRect::new(x, y, w, h))
    }
}

/// A node in the frame scene graph
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "t", rename_all = "lowercase")]
pub enum FrameNode {
    /// Text node with styled runs
    #[serde(rename = "text")]
    Text {
        runs: Vec<TextRun>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        wrap: Option<WrapMode>,
    },
    
    /// Container box with layout properties
    #[serde(rename = "box")]
    Box {
        #[serde(default, skip_serializing_if = "Option::is_none")]
        layout: Option<BoxLayout>,
        children: Vec<FrameNode>,
    },
}

/// A run of text with uniform styling
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TextRun {
    #[serde(rename = "txt")]
    pub text: String,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub style: Option<TextStyle>,
}

/// Text styling attributes
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TextStyle {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub fg: Option<Color>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub bg: Option<Color>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub bold: Option<bool>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub dim: Option<bool>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub italic: Option<bool>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub underline: Option<bool>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub inverse: Option<bool>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub strikethrough: Option<bool>,
}

/// Color specification
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum Color {
    /// Named color (e.g., "red", "blue")
    Named(String),
    /// RGB hex color (e.g., "#ff0000")
    Hex { #[serde(rename = "hex")] hex: String },
    /// ANSI 256 color
    Ansi256 { #[serde(rename = "ansi256")] index: u8 },
}

/// Text wrapping mode
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum WrapMode {
    Wrap,
    Truncate,
    TruncateStart,
    TruncateMiddle,
    None,
}

/// Box layout configuration
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct BoxLayout {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub direction: Option<FlexDirection>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub gap: Option<u16>,
    
    #[serde(default, rename = "px", skip_serializing_if = "Option::is_none")]
    pub padding_x: Option<u16>,
    
    #[serde(default, rename = "py", skip_serializing_if = "Option::is_none")]
    pub padding_y: Option<u16>,
    
    #[serde(default, rename = "mx", skip_serializing_if = "Option::is_none")]
    pub margin_x: Option<u16>,
    
    #[serde(default, rename = "my", skip_serializing_if = "Option::is_none")]
    pub margin_y: Option<u16>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub width: Option<Dimension>,
    
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub height: Option<Dimension>,
    
    #[serde(default, rename = "flexGrow", skip_serializing_if = "Option::is_none")]
    pub flex_grow: Option<u16>,
    
    #[serde(default, rename = "flexShrink", skip_serializing_if = "Option::is_none")]
    pub flex_shrink: Option<u16>,
}

/// Flex container direction
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FlexDirection {
    Row,
    Column,
}

/// Dimension specification
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum Dimension {
    /// Fixed size in cells
    Cells(u16),
    /// Fill available space
    #[serde(rename = "fill")]
    Fill,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_dirty_rect_merge() {
        let r1 = DirtyRect::new(0, 0, 10, 10);
        let r2 = DirtyRect::new(5, 5, 10, 10);
        let merged = r1.merge(&r2).unwrap();
        assert_eq!(merged.x, 0);
        assert_eq!(merged.y, 0);
        assert_eq!(merged.width, 15);
        assert_eq!(merged.height, 15);
    }
    
    #[test]
    fn test_dirty_rect_no_merge() {
        let r1 = DirtyRect::new(0, 0, 5, 5);
        let r2 = DirtyRect::new(10, 10, 5, 5);
        assert!(r1.merge(&r2).is_none());
    }
    
    #[test]
    fn test_frame_protocol_serialization() {
        let frame = FrameProtocol {
            version: SCHEMA_VERSION,
            sequence: 1,
            width: 80,
            height: 24,
            root: FrameNode::Text {
                runs: vec![TextRun {
                    text: "Hello".to_string(),
                    style: Some(TextStyle {
                        bold: Some(true),
                        ..Default::default()
                    }),
                }],
                wrap: None,
            },
            dirty_rects: None,
        };
        
        let json = serde_json::to_string(&frame).unwrap();
        let decoded: FrameProtocol = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.version, SCHEMA_VERSION);
        assert_eq!(decoded.sequence, 1);
    }
}
