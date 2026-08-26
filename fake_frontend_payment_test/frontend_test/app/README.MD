Estrutura de rotas (App Router)
Crie as rotas abaixo conforme as User Stories forem entrando na Sprint.

exemplo de um contexto diferente de como se é aplicado (devemos modificar para o contexto da nossa aplicação de pagamento)

src/app/
├── layout.tsx                    # MantineProvider + ColorSchemeScript (ja pronto)
├── page.tsx                      # landing (ja pronto)
├── login/page.tsx                # US-00
├── primeiro-acesso/page.tsx      # US-00 (ativar login / auto-cadastro)
├── (paciente)/
│   ├── layout.tsx                # guarda: tipo_usuario === 'PACIENTE'
│   ├── meus-dados/page.tsx       # US-04
│   ├── medicos/page.tsx          # US-06
│   ├── agendar/page.tsx          # US-07 + US-08
│   └── consultas/
│       ├── page.tsx              # US-10
│       └── [id]/page.tsx         # US-10 (detalhe) + US-11 (cancelar)
└── (atendente)/
    ├── layout.tsx                # guarda: tipo_usuario === 'ATENDENTE'
    ├── pacientes/page.tsx        # US-03
    ├── especialidades/page.tsx   # US-01
    ├── medicos/page.tsx          # US-02
    ├── agenda/page.tsx           # US-05
    └── consultas/page.tsx        # US-09, US-12, US-13
A tela de login vive em src/app/login/page.tsx (formulário + chamada à API na própria rota, com o visual compartilhado em src/components/AuthShell.tsx). A guarda de perfil fica em src/components/auth/AuthGuard.tsx, e a home de cada perfil em src/lib/rotas.ts — a mesma constante que o proxy.ts lê, para login e guarda nunca discordarem.