<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue?: string | number
  type?: string
  placeholder?: string
  disabled?: boolean
  readonly?: boolean
  error?: string
  icon?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  type: 'text',
  placeholder: '',
  disabled: false,
  readonly: false,
  error: '',
  icon: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'focus', event: FocusEvent): void
  (e: 'blur', event: FocusEvent): void
}>()

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}

const inputClasses = computed(() => [
  'app-input',
  {
    'app-input--with-icon': props.icon,
    'app-input--error': !!props.error,
    'app-input--disabled': props.disabled,
  },
])
</script>

<template>
  <div class="app-input-wrapper">
    <div :class="inputClasses">
      <span v-if="icon || $slots.icon" class="app-input__icon">
        <slot name="icon">
          <!-- Fallback search SVG icon matching Figma -->
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
        </slot>
      </span>
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        class="app-input__field"
        @input="handleInput"
        @focus="emit('focus', $event)"
        @blur="emit('blur', $event)"
      />
      <span v-if="$slots.suffix" class="app-input__suffix">
        <slot name="suffix" />
      </span>
    </div>
    <span v-if="error" class="app-input__error-msg">{{ error }}</span>
  </div>
</template>

<style scoped>
.app-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
}

.app-input {
  display: flex;
  align-items: center;
  background-color: rgba(15, 23, 42, 0.6);
  border: 1px solid var(--color-neutral-700);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  transition: all var(--transition-fast);
  color: var(--text-main);
  position: relative;
}

.app-input:focus-within {
  border-color: var(--color-primary-peach);
  box-shadow: 0 0 0 2px rgba(252, 165, 165, 0.15);
  background-color: var(--color-neutral-850);
}

.app-input--error {
  border-color: var(--color-primary-500) !important;
}

.app-input--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.app-input__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
  color: var(--color-neutral-400);
}

.app-input__field {
  width: 100%;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-main);
  font-family: var(--font-mono);
  font-size: 0.875rem;
  letter-spacing: -0.01em;
}

.app-input__field::placeholder {
  color: var(--color-neutral-500);
  font-family: var(--font-mono);
}

.app-input__suffix {
  margin-left: 8px;
  display: flex;
  align-items: center;
}

.app-input__error-msg {
  font-family: var(--font-sans);
  font-size: 0.75rem;
  color: #f87171;
  padding-left: 2px;
}
</style>
