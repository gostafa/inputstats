// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package windows

// Raw Input flag constants (winuser.h). Kept here so parsers are OS-independent.
// RI_MOUSE_LEFT_BUTTON_DOWN shares value 0x01 with RI_KEY_BREAK — reuse riKeyBreak.
const (
	riKeyBreak = 0x01

	riMouseRightButtonDown = 0x0004

	flagClear = 0
)
