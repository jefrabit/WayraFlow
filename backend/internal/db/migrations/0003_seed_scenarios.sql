-- Predefined scenarios from README section 6. Every value is an assumption:
-- there are no public traffic counts for Juliaca.
INSERT INTO scenarios (id, name, description, demand) VALUES
 ('base', 'Tráfico base', 'Hora de análisis con la demanda observada como supuesto.',
  '{"corridors":{"C1":700,"C2":600,"C3":650,"N8":400},"mix":{"L5":0.35,"M1":0.35,"M2":0.20,"N":0.10}}'::jsonb),
 ('hora-punta-regional', 'Hora punta entrada/salida regional', 'Pico de acceso a la ciudad por los tres corredores.',
  '{"corridors":{"C1":1000,"C2":750,"C3":1000,"N8":650},"mix":{"L5":0.25,"M1":0.30,"M2":0.30,"N":0.15}}'::jsonb),
 ('dia-de-feria', 'Día de feria (domingo)', 'Mayor peso de mototaxis y flujo hacia el mercado.',
  '{"corridors":{"C1":850,"C2":1050,"C3":800,"N8":450},"mix":{"L5":0.45,"M1":0.25,"M2":0.20,"N":0.10}}'::jsonb)
ON CONFLICT (id) DO NOTHING;
