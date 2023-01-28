import { h, JSX } from "preact";
import {
  Anchor,
  AppShell,
  Flex,
  Group,
  Header,
  Navbar,
  MantineProvider,
  Text,
  ThemeIcon,
  Title,
  UnstyledButton,
  useMantineTheme,
} from "@mantine/core";
import { IconSearch, IconSettings } from "@tabler/icons";

interface MainLinkProps {
  iconSvg: string;
  href: string;
  label: string;
}

const MainLink = ({ iconSvg, href, label }: MainLinkProps) => {
  return (
    <UnstyledButton
      component="a"
      href={href}
      sx={(theme) => ({
        display: "block",
        width: "100%",
        padding: theme.spacing.xs,
        borderRadius: theme.radius.sm,
        color:
          theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.black,

        "&:hover": {
          backgroundColor:
            theme.colorScheme === "dark"
              ? theme.colors.dark[6]
              : theme.colors.gray[0],
        },
      })}
    >
      <Group>
        <ThemeIcon
          variant="light"
          dangerouslySetInnerHTML={{ __html: iconSvg }}
        ></ThemeIcon>

        <Text size="sm">{label}</Text>
      </Group>
    </UnstyledButton>
  );
};

export const LogsuckAppShell = (props: any) => {
  const theme = useMantineTheme();
  return (
    <MantineProvider withGlobalStyles>
      <div id="app">
        <AppShell
          padding="md"
          navbar={
            <MantineProvider theme={{ colorScheme: "dark" }}>
              <Navbar width={{ base: 160 }} top={0}>
                <Navbar.Section>
                  <Title>
                    <Anchor href="/" px="xs">
                      Logsuck
                    </Anchor>
                  </Title>
                </Navbar.Section>
                <Navbar.Section>
                  <MainLink
                    label="Search"
                    iconSvg={(IconSearch as any)()}
                    href="/search"
                  ></MainLink>
                  <MainLink
                    label="Config"
                    iconSvg={(IconSettings as any)()}
                    href="/config"
                  ></MainLink>
                </Navbar.Section>
              </Navbar>
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
