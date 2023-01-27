import { h } from "preact";
import {
  Anchor,
  AppShell,
  Flex,
  Header,
  MantineProvider,
  Title,
  useMantineTheme,
} from "@mantine/core";

export const LogsuckAppShell = (props: any) => {
  const theme = useMantineTheme();
  return (
    <MantineProvider
      withNormalizeCSS
      withGlobalStyles
      theme={{
        focusRingStyles: {
          // reset styles are applied to <button /> and <a /> elements
          // in &:focus:not(:focus-visible) selector to mimic
          // default browser behavior for native <button /> and <a /> elements
          resetStyles: () => ({ outline: "none" }),

          // styles applied to all elements expect inputs based on Input component
          // styled are added with &:focus selector
          styles: (theme) => ({
            outline: `2px solid ${theme.colors.orange[5]}`,
          }),

          // focus styles applied to components that are based on Input
          // styled are added with &:focus selector
          inputStyles: (theme) => ({
            outline: `2px solid ${theme.colors.orange[5]}`,
          }),
        },
      }}
    >
      <div id="app">
        <AppShell
          padding="md"
          header={
            <MantineProvider theme={{ colorScheme: "dark" }}>
              <Header
                height={60}
                withBorder
                px="md"
                style={{ background: theme.colors.dark[8] }}
              >
                <Flex direction="row" justify="space-between" align="center">
                  <Title>
                    <Anchor href="/">logsuck</Anchor>
                  </Title>
                  <Flex direction="row" gap="md">
                    <Anchor href="/search">Search</Anchor>
                    <Anchor href="/config">Config</Anchor>
                  </Flex>
                </Flex>
              </Header>
            </MantineProvider>
          }
          styles={(theme: any) => ({
            main: {
              backgroundColor:
                theme.colorScheme === "dark"
                  ? theme.colors.dark[8]
                  : theme.colors.gray[0],
            },
          })}
        >
          {props.children}
        </AppShell>
      </div>
    </MantineProvider>
  );
};
