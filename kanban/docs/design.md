# MAX visual language

TaskFlow uses the production MAX messenger and MAX UI tokens as its visual source of truth.
Custom class names are preserved, but colors, typography, spacing, radii, controls,
and platform behavior follow MAX.

## Core color tokens

| Token | Light | Dark |
|---|---|---|
| Application surface | `#EDEEF2` | `#0F0F12` |
| Primary surface | `#FFFFFF` | `#17181C` |
| Card surface | `#FFFFFF` | `#25262D` |
| Secondary surface | `#F5F7FA` | `#25262D` |
| Primary action | `#007AFF` | `#007AFF` |
| Action hover | `#479FFF` | `#479FFF` |
| Action pressed | `#006EE5` | `#006EE5` |
| Primary text | `#060708` | `#FFFFFF` |
| Secondary text | `rgba(6, 7, 8, 0.68)` | `rgba(255, 255, 255, 0.8)` |
| Tertiary text | `rgba(6, 7, 8, 0.52)` | `rgba(255, 255, 255, 0.64)` |
| Primary divider | `rgba(12, 13, 14, 0.16)` | `rgba(255, 255, 255, 0.12)` |
| Secondary divider | `rgba(12, 13, 14, 0.06)` | `rgba(255, 255, 255, 0.06)` |
| Success | `#1ABE43` | `#2BC644` |
| Error | `#FF303C` | `#CE4257` |
| Modal overlay | `rgba(12, 13, 14, 0.32)` | `rgba(13, 13, 13, 0.64)` |

Cards use flat surfaces and thin dividers instead of decorative shadows.
The large modal shadow follows the MAX modal elevation treatment.

## Typography and controls

- Base text: Roboto, 16/20.
- Titles: 17/24 or 24/28 semibold.
- Labels: 12/16 semibold, uppercase, `0.3px` spacing.
- Primary buttons: 40px height, 12px radius, 14/20 semibold.
- Small buttons: 32px height, 8px radius, 13px semibold.
- Secondary buttons use filled neutral surfaces.
- Ghost actions use transparent backgrounds and themed text.
- Inputs use filled neutral surfaces and themed focus rings.
- Checkboxes use native controls with the MAX accent color.
- Spinners use neutral tertiary styling rather than colored branding.
- Tap highlights are disabled for native-feeling touch interaction.

## Platform behavior

The frontend reads `window.WebApp.platform` and stores it as
`document.documentElement.dataset.maxPlatform`.

- `ios`: system font stack and tighter letter spacing.
- `android`: Roboto and standard MAX letter spacing.
- `desktop` and `web`: desktop typography baseline.

Hover styling is scoped to `(hover: hover) and (pointer: fine)`.
Touch controls use `:active` states, manipulation touch actions, and safe-area padding.

## Component mapping

- Application shell: MAX secondary surface.
- Navigation: neutral sidebar/list rows with card-like active state.
- Organizations and boards: white island cards with 16px radii.
- Kanban columns: filled secondary containers.
- Tasks: white 16px message-style cards with dividers.
- Avatars: circular initials with translucent themed backgrounds.
- Counters: neutral pills, with themed active mobile counts.
- Modals: 20px cards on mobile bottom sheets.
- Icons: outline Lucide icons; no emoji status or navigation symbols.
