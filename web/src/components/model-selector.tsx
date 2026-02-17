import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '~/components/ui/select'

interface Model {
  id: string
  name: string
  input_cost_per_1k: number
  output_cost_per_1k: number
}

const defaultModels: Model[] = [
  {
    id: 'anthropic/claude-sonnet-4',
    name: 'Claude Sonnet 4',
    input_cost_per_1k: 0.003,
    output_cost_per_1k: 0.015,
  },
  {
    id: 'anthropic/claude-haiku-4',
    name: 'Claude Haiku 4',
    input_cost_per_1k: 0.0008,
    output_cost_per_1k: 0.004,
  },
  { id: 'openai/gpt-4o', name: 'GPT-4o', input_cost_per_1k: 0.0025, output_cost_per_1k: 0.01 },
  {
    id: 'openai/gpt-4o-mini',
    name: 'GPT-4o Mini',
    input_cost_per_1k: 0.00015,
    output_cost_per_1k: 0.0006,
  },
  {
    id: 'google/gemini-2.5-pro',
    name: 'Gemini 2.5 Pro',
    input_cost_per_1k: 0.00125,
    output_cost_per_1k: 0.01,
  },
]

interface ModelSelectorProps {
  value: string
  onChange: (value: string) => void
  models?: Model[]
}

export function ModelSelector({ value, onChange, models = defaultModels }: ModelSelectorProps) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-7 w-[180px] border-neutral-800 bg-neutral-900 text-[10px] text-neutral-300">
        <SelectValue placeholder="Select model" />
      </SelectTrigger>
      <SelectContent>
        {models.map((model) => (
          <SelectItem key={model.id} value={model.id} className="text-[10px]">
            <div className="flex items-center justify-between gap-4">
              <span>{model.name}</span>
              <span className="text-muted-foreground">
                ${((model.input_cost_per_1k + model.output_cost_per_1k) / 2).toFixed(4)}/1k
              </span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
